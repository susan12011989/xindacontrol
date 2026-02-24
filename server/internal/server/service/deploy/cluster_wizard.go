package deploy

import (
	"fmt"
	"os"
	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	cloudaws "server/internal/server/service/cloud_aws"
	utilSvc "server/internal/server/service/utils"
	"server/internal/server/utils"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"xorm.io/xorm"
)

const wizardTotalSteps = 8

// ClusterWizard 集群一站式创建向导（SSE 流式）
// 支持恢复模式：req.MerchantId > 0 时跳过已完成的步骤
func ClusterWizard(req model.ClusterWizardReq, operator string, progress func(model.ClusterWizardStep)) error {
	// 设置默认值
	if req.DbInstanceType == "" {
		req.DbInstanceType = "r5.large"
	}
	if req.DbVolumeSizeGiB == 0 {
		req.DbVolumeSizeGiB = 300
	}
	if req.MinioInstanceType == "" {
		req.MinioInstanceType = "t3.medium"
	}
	if req.MinioVolumeSizeGiB == 0 {
		req.MinioVolumeSizeGiB = 200
	}
	if req.AppInstanceType == "" {
		req.AppInstanceType = "t3.large"
	}
	if req.AppVolumeSizeGiB == 0 {
		req.AppVolumeSizeGiB = 30
	}
	if req.KeyName == "" {
		req.KeyName = "tsdd-deploy-key" // 集群默认 Key Pair
	}

	// ========== Step 1: 创建商户（恢复模式跳过） ==========
	var merchantId int
	// existingServers: role -> *entity.Servers (恢复模式时已注册的服务器)
	existingServers := make(map[string]*entity.Servers)
	// existingDeployed: role -> bool (恢复模式时已部署的节点)
	existingDeployed := make(map[string]bool)

	if req.MerchantId > 0 {
		// ===== 恢复模式 =====
		merchantId = req.MerchantId
		var merchant entity.Merchants
		has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&merchant)
		if err != nil || !has {
			sendProgress(progress, 1, "failed", "创建商户记录", fmt.Sprintf("商户 ID %d 不存在", merchantId))
			return fmt.Errorf("商户 ID %d 不存在", merchantId)
		}
		// 设置 MerchantName（恢复模式可能没传）
		if req.MerchantName == "" {
			req.MerchantName = merchant.Name
		}
		sendProgressWithMerchant(progress, 1, "success", "创建商户记录", fmt.Sprintf("恢复模式, 商户 ID: %d (%s)", merchantId, merchant.Name), merchantId)

		// 查询已注册的集群服务器（按名称后缀匹配角色）
		var servers []entity.Servers
		dbs.DBAdmin.Where("merchant_id = ? AND aws_instance_id != ''", merchantId).Find(&servers)
		for i := range servers {
			s := &servers[i]
			for _, role := range []string{"db", "minio", "app"} {
				if strings.HasSuffix(s.Name, "-"+role) {
					existingServers[role] = s
					break
				}
			}
		}

		// 查询已部署的节点
		var nodes []entity.ClusterNodes
		dbs.DBAdmin.Where("merchant_id = ? AND status = ?", merchantId, entity.ClusterStatusDeployed).Find(&nodes)
		for _, n := range nodes {
			existingDeployed[n.NodeRole] = true
		}

		logx.Infof("[ClusterWizard] 恢复模式: merchantId=%d, 已有服务器=%v, 已部署=%v",
			merchantId, mapKeys(existingServers), mapKeys2(existingDeployed))
	} else {
		// ===== 新建模式 =====
		if req.MerchantName == "" {
			sendProgress(progress, 1, "failed", "创建商户记录", "商户名称不能为空")
			return fmt.Errorf("商户名称不能为空")
		}
		if req.Port == 0 {
			sendProgress(progress, 1, "failed", "创建商户记录", "端口不能为空")
			return fmt.Errorf("端口不能为空")
		}
		if req.CloudAccountId == 0 {
			sendProgress(progress, 1, "failed", "创建商户记录", "请选择 AWS 云账号")
			return fmt.Errorf("CloudAccountId 不能为空")
		}

		sendProgress(progress, 1, "running", "创建商户记录", "")
		var err error
		merchantId, err = createMerchantForCluster(req)
		if err != nil {
			sendProgress(progress, 1, "failed", "创建商户记录", fmt.Sprintf("失败: %v", err))
			return fmt.Errorf("创建商户失败: %v", err)
		}
		sendProgressWithMerchant(progress, 1, "success", "创建商户记录", fmt.Sprintf("商户 ID: %d", merchantId), merchantId)
	}

	// 恢复模式：自动获取 CloudAccountId 和 RegionId
	if req.CloudAccountId == 0 {
		var acc entity.CloudAccounts
		has, _ := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Get(&acc)
		if has {
			req.CloudAccountId = int64(acc.Id)
		}
	}
	if req.RegionId == "" {
		// 从已有服务器获取 region
		for _, s := range existingServers {
			if s.AwsRegionId != "" {
				req.RegionId = s.AwsRegionId
				break
			}
		}
		if req.RegionId == "" {
			req.RegionId = "ap-east-1" // 默认
		}
	}

	// 获取 AWS 云账号
	acc, err := awscloud.ResolveAwsAccount(nil, merchantId, req.CloudAccountId)
	if err != nil {
		sendProgress(progress, 2, "failed", "创建 DB EC2 实例", fmt.Sprintf("获取 AWS 账号失败: %v", err))
		return fmt.Errorf("获取 AWS 账号失败: %v", err)
	}

	// 读取 SSH 私钥（EC2 使用 Key Pair 认证，不支持密码）
	var sshPrivateKey string
	if req.KeyName != "" {
		keyPaths := []string{
			fmt.Sprintf("/home/ubuntu/.ssh/%s.pem", req.KeyName),
			fmt.Sprintf("%s/.ssh/%s.pem", os.Getenv("HOME"), req.KeyName),
		}
		for _, kp := range keyPaths {
			if data, err := os.ReadFile(kp); err == nil {
				sshPrivateKey = string(data)
				logx.Infof("[ClusterWizard] 读取 SSH 私钥: %s (%d bytes)", kp, len(data))
				break
			}
		}
		if sshPrivateKey == "" {
			sendProgress(progress, 2, "failed", "创建 DB EC2 实例",
				fmt.Sprintf("找不到 SSH 私钥文件: %s.pem (请放置在 ~/.ssh/ 目录)", req.KeyName))
			return fmt.Errorf("找不到 SSH 私钥: %s.pem", req.KeyName)
		}
	}

	// ========== Steps 2-4: 并行创建 EC2（跳过已存在的） ==========
	type ec2Result struct {
		role       string
		instanceId string
		publicIP   string
		err        error
		skipped    bool // 恢复模式跳过
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make(map[string]*ec2Result)
	)

	ec2Configs := []struct {
		step         int
		role         string
		amiId        string
		instanceType string
		volumeSize   int32
	}{
		{2, "db", req.DbAmiId, req.DbInstanceType, req.DbVolumeSizeGiB},
		{3, "minio", req.MinioAmiId, req.MinioInstanceType, req.MinioVolumeSizeGiB},
		{4, "app", req.AppAmiId, req.AppInstanceType, req.AppVolumeSizeGiB},
	}

	for _, cfg := range ec2Configs {
		title := fmt.Sprintf("创建 %s EC2 实例", roleLabel(cfg.role))

		// 恢复模式：跳过已有服务器的角色
		if existing, ok := existingServers[cfg.role]; ok {
			results[cfg.role] = &ec2Result{
				role:       cfg.role,
				instanceId: existing.AwsInstanceId,
				publicIP:   existing.Host,
				skipped:    true,
			}
			sendProgress(progress, cfg.step, "success", title,
				fmt.Sprintf("已存在, IP: %s, 实例: %s (跳过)", existing.Host, existing.AwsInstanceId))
			continue
		}

		wg.Add(1)
		go func(step int, role, amiId, instType string, volSize int32) {
			defer wg.Done()
			title := fmt.Sprintf("创建 %s EC2 实例", roleLabel(role))
			amiInfo := amiId
			if amiInfo == "" {
				amiInfo = "默认 Ubuntu"
			}
			sendProgress(progress, step, "running", title, fmt.Sprintf("AMI: %s, 机型: %s, 磁盘: %dGB", amiInfo, instType, volSize))

			serverName := fmt.Sprintf("%s-%s", req.MerchantName, role)
			createReq := model.AwsCreateEc2InstanceReq{
				MerchantId:    merchantId,
				RegionId:      req.RegionId,
				ImageId:       amiId,
				InstanceType:  instType,
				SubnetId:      req.SubnetId,
				KeyName:       req.KeyName,
				VolumeSizeGiB: volSize,
				InstanceName:  serverName,
			}

			instanceId, createErr := cloudaws.CreateEc2Instance(acc, createReq)
			if createErr != nil {
				mu.Lock()
				results[role] = &ec2Result{role: role, err: createErr}
				mu.Unlock()
				sendProgress(progress, step, "failed", title, fmt.Sprintf("创建失败: %v", createErr))
				return
			}

			sendProgress(progress, step, "running", title, fmt.Sprintf("实例 %s 已创建, 等待启动...", instanceId))
			if waitErr := cloudaws.WaitForInstanceRunning(acc, req.RegionId, instanceId, 5*time.Minute); waitErr != nil {
				mu.Lock()
				results[role] = &ec2Result{role: role, instanceId: instanceId, err: waitErr}
				mu.Unlock()
				sendProgress(progress, step, "failed", title, fmt.Sprintf("等待启动超时: %v", waitErr))
				return
			}

			time.Sleep(30 * time.Second)

			publicIP, ipErr := cloudaws.GetInstancePublicIP(acc, req.RegionId, instanceId)
			if ipErr != nil {
				mu.Lock()
				results[role] = &ec2Result{role: role, instanceId: instanceId, err: ipErr}
				mu.Unlock()
				sendProgress(progress, step, "failed", title, fmt.Sprintf("获取 IP 失败: %v", ipErr))
				return
			}

			mu.Lock()
			results[role] = &ec2Result{role: role, instanceId: instanceId, publicIP: publicIP}
			mu.Unlock()
			sendProgress(progress, step, "success", title, fmt.Sprintf("IP: %s, 实例: %s", publicIP, instanceId))
		}(cfg.step, cfg.role, cfg.amiId, cfg.instanceType, cfg.volumeSize)
	}
	wg.Wait()

	// 统计成功/失败
	var failedRoles []string
	for _, role := range []string{"db", "minio", "app"} {
		r := results[role]
		if r == nil || r.err != nil {
			failedRoles = append(failedRoles, role)
		}
	}
	hasFailure := len(failedRoles) > 0

	// ========== Step 5: 注册服务器（仅注册成功的、尚未注册的） ==========
	sendProgress(progress, 5, "running", "注册服务器", "")

	serverIds := make(map[string]int)
	// 先填充已有的
	for role, s := range existingServers {
		serverIds[role] = s.Id
	}

	var registerCount int
	for _, role := range []string{"db", "minio", "app"} {
		r := results[role]
		if r == nil || r.err != nil {
			continue // EC2 创建失败的跳过
		}
		if r.skipped {
			continue // 已注册的跳过
		}
		server := &entity.Servers{
			ServerType:    1,
			MerchantId:    merchantId,
			Name:          fmt.Sprintf("%s-%s", req.MerchantName, role),
			Host:          r.publicIP,
			Port:          22,
			Username:      "ubuntu",
			AuthType:      2,
			PrivateKey:    sshPrivateKey,
			AwsInstanceId:   r.instanceId,
			AwsRegionId:     req.RegionId,
			CloudType:       "aws",
			CloudInstanceId: r.instanceId,
			CloudRegionId:   req.RegionId,
			Status:        1,
			Description:   fmt.Sprintf("集群%s节点-自动创建", roleLabel(role)),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if _, err := dbs.DBAdmin.Insert(server); err != nil {
			logx.Errorf("[ClusterWizard] 注册 %s 服务器失败: %v", role, err)
			continue
		}
		serverIds[role] = server.Id
		registerCount++
		logx.Infof("[ClusterWizard] 注册 %s 服务器: id=%d, host=%s", role, server.Id, r.publicIP)
	}

	if hasFailure {
		msg := fmt.Sprintf("已注册 %d 台 (%s EC2 创建失败)", registerCount, strings.Join(failedRoles, ", "))
		sendProgress(progress, 5, "success", "注册服务器", msg)
	} else {
		sendProgress(progress, 5, "success", "注册服务器",
			fmt.Sprintf("DB=%d, MinIO=%d, App=%d", serverIds["db"], serverIds["minio"], serverIds["app"]))
	}

	// 等待 SSH 就绪 + 检测内网 IP（仅对有 EC2 的角色）
	privateIPs := make(map[string]string)
	for _, role := range []string{"db", "minio", "app"} {
		r := results[role]
		if r == nil || r.err != nil || r.publicIP == "" {
			continue
		}
		if !r.skipped {
			// 新创建的实例需要等待 SSH，通知前端
			stepIdx := map[string]int{"db": 6, "minio": 7, "app": 8}
			sendProgress(progress, stepIdx[role], "running", fmt.Sprintf("部署 %s 节点", roleLabel(role)),
				fmt.Sprintf("等待 SSH 就绪: %s ...", r.publicIP))
			if !waitSSHReadyWithKey(r.publicIP, "ubuntu", "", sshPrivateKey, 3*time.Minute) {
				logx.Errorf("[ClusterWizard] SSH 连接 %s (%s) 超时，继续尝试...", role, r.publicIP)
			}
		}
		client := &utils.SSHClient{
			Host:       r.publicIP,
			Port:       22,
			Username:   "ubuntu",
			PrivateKey: sshPrivateKey,
		}
		if err := client.Connect(); err == nil {
			if pip := DetectPrivateIP(client); pip != "" {
				privateIPs[role] = pip
				logx.Infof("[ClusterWizard] %s 内网 IP: %s", role, pip)
			}
			client.Close()
		}
	}

	// 确定内网 IP（回退到公网）
	dbPrivateIP := privateIPs["db"]
	minioPrivateIP := privateIPs["minio"]
	if dbPrivateIP == "" {
		if r := results["db"]; r != nil && r.err == nil {
			dbPrivateIP = r.publicIP
		}
	}
	if minioPrivateIP == "" {
		if r := results["minio"]; r != nil && r.err == nil {
			minioPrivateIP = r.publicIP
		}
	}

	// ========== Steps 6-8: 部署节点（跳过缺失的/已部署的） ==========
	deployConfigs := []struct {
		step     int
		role     string
		title    string
		makeReq  func() model.DeployNodeReq
		successMsg func() string
	}{
		{
			step: 6, role: "db", title: "部署 DB 节点",
			makeReq: func() model.DeployNodeReq {
				return model.DeployNodeReq{
					ServerId: serverIds["db"], MerchantId: merchantId,
					NodeRole: "db", ForceReset: false,
				}
			},
			successMsg: func() string {
				return fmt.Sprintf("MySQL + Redis 部署成功, 内网: %s", dbPrivateIP)
			},
		},
		{
			step: 7, role: "minio", title: "部署 MinIO 节点",
			makeReq: func() model.DeployNodeReq {
				return model.DeployNodeReq{
					ServerId: serverIds["minio"], MerchantId: merchantId,
					NodeRole: "minio", ForceReset: false,
				}
			},
			successMsg: func() string {
				return fmt.Sprintf("MinIO 部署成功, 内网: %s", minioPrivateIP)
			},
		},
		{
			step: 8, role: "app", title: "部署 App 节点",
			makeReq: func() model.DeployNodeReq {
				return model.DeployNodeReq{
					ServerId: serverIds["app"], MerchantId: merchantId,
					NodeRole: "app", ForceReset: false,
					DBHost: dbPrivateIP, MinioHost: minioPrivateIP, WKNodeId: 1001,
				}
			},
			successMsg: func() string {
				return "WuKongIM + tsdd-server + web + manager 部署成功"
			},
		},
	}

	var deployFailures []string
	for _, dc := range deployConfigs {
		// 已部署的跳过
		if existingDeployed[dc.role] {
			sendProgress(progress, dc.step, "success", dc.title, "已部署, 跳过")
			continue
		}

		// 没有服务器的跳过（EC2 创建失败）
		if serverIds[dc.role] == 0 {
			sendProgress(progress, dc.step, "skipped", dc.title, "EC2 未创建, 跳过")
			deployFailures = append(deployFailures, dc.role)
			continue
		}

		// App 节点依赖 DB，如果 DB 未就绪则跳过
		if dc.role == "app" && dbPrivateIP == "" {
			sendProgress(progress, dc.step, "skipped", dc.title, "DB 节点未就绪, 无法部署 App")
			deployFailures = append(deployFailures, dc.role)
			continue
		}

		r := results[dc.role]
		ip := ""
		if r != nil {
			ip = r.publicIP
		}
		sendProgress(progress, dc.step, "running", dc.title, fmt.Sprintf("服务器: %s", ip))

		deployReq := dc.makeReq()
		resp, err := DeployNodeByServerId(deployReq, operator)
		if err != nil || !resp.Success {
			errMsg := "部署失败"
			if err != nil {
				errMsg = err.Error()
			} else if resp.Message != "" {
				errMsg = resp.Message
			}
			sendProgress(progress, dc.step, "failed", dc.title, errMsg)
			deployFailures = append(deployFailures, dc.role)
			// 不 return，继续部署其他节点
			continue
		}
		sendProgress(progress, dc.step, "success", dc.title, dc.successMsg())
	}

	// 更新商户 server_ip（App 公网 IP）
	if r := results["app"]; r != nil && r.err == nil && r.publicIP != "" {
		if _, err := dbs.DBAdmin.Where("id = ?", merchantId).Cols("server_ip").Update(&entity.Merchants{ServerIP: r.publicIP}); err != nil {
			logx.Errorf("[ClusterWizard] 更新商户 server_ip 失败: %v", err)
		}
	}

	// 汇总结果
	allFailures := append(failedRoles, deployFailures...)
	// 去重
	seen := make(map[string]bool)
	var uniqueFailures []string
	for _, f := range allFailures {
		if !seen[f] {
			seen[f] = true
			uniqueFailures = append(uniqueFailures, f)
		}
	}

	if len(uniqueFailures) > 0 {
		logx.Infof("[ClusterWizard] 部分完成: merchant=%s(id=%d), 失败角色=%v", req.MerchantName, merchantId, uniqueFailures)
		return fmt.Errorf("%s 节点未完成，可点击「重试」补部署", strings.Join(uniqueFailures, ", "))
	}

	logx.Infof("[ClusterWizard] 集群部署完成: merchant=%s(id=%d), db=%s, minio=%s, app=%s",
		req.MerchantName, merchantId,
		results["db"].publicIP, results["minio"].publicIP, results["app"].publicIP)

	return nil
}

// sendProgress 发送进度通知
func sendProgress(callback func(model.ClusterWizardStep), step int, status, title, message string) {
	callback(model.ClusterWizardStep{
		Step:    step,
		Total:   wizardTotalSteps,
		Title:   title,
		Status:  status,
		Message: message,
	})
}

// sendProgressWithMerchant 发送进度通知（附带 MerchantId，供前端重试时使用）
func sendProgressWithMerchant(callback func(model.ClusterWizardStep), step int, status, title, message string, merchantId int) {
	callback(model.ClusterWizardStep{
		Step:       step,
		Total:      wizardTotalSteps,
		Title:      title,
		Status:     status,
		Message:    message,
		MerchantId: merchantId,
	})
}

// roleLabel 角色中文名
func roleLabel(role string) string {
	switch role {
	case "db":
		return "DB"
	case "minio":
		return "MinIO"
	case "app":
		return "App"
	default:
		return role
	}
}

// mapKeys 提取 map 的 key 列表（调试日志用）
func mapKeys(m map[string]*entity.Servers) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func mapKeys2(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// waitSSHReady 等待 SSH 可连接
func waitSSHReady(host, username, password string, timeout time.Duration) bool {
	return waitSSHReadyWithKey(host, username, password, "", timeout)
}

func waitSSHReadyWithKey(host, username, password, privateKey string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		client := &utils.SSHClient{
			Host:       host,
			Port:       22,
			Username:   username,
			Password:   password,
			PrivateKey: privateKey,
		}
		if err := client.Connect(); err == nil {
			client.Close()
			return true
		}
		time.Sleep(10 * time.Second)
	}
	return false
}

// createMerchantForCluster 为集群向导创建商户记录（内联实现，避免循环依赖）
func createMerchantForCluster(req model.ClusterWizardReq) (int, error) {
	if req.Port < 10000 || req.Port > 65535 {
		return 0, fmt.Errorf("port 必须在 10000-65535 之间")
	}

	now := time.Now()
	expiredAt := now.AddDate(0, 0, 30)
	if req.ExpiredAt != "" {
		parsed, err := time.ParseInLocation(time.DateTime, req.ExpiredAt, time.Local)
		if err == nil {
			expiredAt = parsed
		}
	}

	no := utilSvc.Port2Enterprise(uint16(req.Port))

	merchant := &entity.Merchants{
		No:        no,
		ServerIP:  "pending-cluster-deploy",
		Port:      req.Port,
		Name:      req.MerchantName,
		AppName:   req.AppName,
		Status:    1,
		ExpiredAt: expiredAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var sourceAccount entity.CloudAccounts
	has, err := dbs.DBAdmin.Where("id = ? AND status = 1", req.CloudAccountId).Get(&sourceAccount)
	if err != nil || !has {
		return 0, fmt.Errorf("AWS 账号不存在或不可用")
	}

	if err := dbs.DBAdmin.WithTx(func(session *xorm.Session) error {
		if _, err := session.Insert(merchant); err != nil {
			return err
		}

		merchantServer := &entity.Servers{
			ServerType:  1,
			MerchantId:  merchant.Id,
			Name:        fmt.Sprintf("%s-商户服务器", merchant.Name),
			Host:        "pending-cluster-deploy",
			Port:        consts.DefaultPort,
			Username:    consts.DefaultUsername,
			AuthType:    1,
			Password:    consts.DefaultPassword,
			DeployPath:  consts.DeployPath,
			Status:      1,
			Description: "商户服务器-集群向导自动创建",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if _, err := session.Insert(merchantServer); err != nil {
			return err
		}

		account := entity.CloudAccounts{
			AccountType:     "merchant",
			MerchantId:      merchant.Id,
			Name:            fmt.Sprintf("%s-aws", merchant.Name),
			CloudType:       "aws",
			AccessKeyId:     sourceAccount.AccessKeyId,
			AccessKeySecret: sourceAccount.AccessKeySecret,
			Description:     fmt.Sprintf("商户:%s (集群向导创建)", merchant.Name),
			Status:          1,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if _, err := session.Insert(&account); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return merchant.Id, nil
}
