package deploy

import (
	"fmt"
	"server/internal/server/model"
	awscloud "server/internal/server/cloud/aws"
	cloudaws "server/internal/server/service/cloud_aws"
	utilSvc "server/internal/server/service/utils"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"xorm.io/xorm"
)

const clusterWizardTotalSteps = 8

// ClusterWizard 集群向导：一键创建商户 + 3台EC2 + 部署
// progress 回调发送流式进度
func ClusterWizard(req model.ClusterWizardReq, operator string, progress func(step model.ClusterWizardStepResp)) error {
	var merchantId int

	// ========== Step 1: 创建商户 ==========
	if req.MerchantId > 0 {
		// 恢复模式
		merchantId = req.MerchantId
		progress(model.ClusterWizardStepResp{
			Step: 1, Total: clusterWizardTotalSteps,
			Title: "创建商户记录", Status: "skipped",
			Message: fmt.Sprintf("使用已有商户 ID: %d", merchantId),
			MerchantId: merchantId,
		})
	} else {
		progress(model.ClusterWizardStepResp{
			Step: 1, Total: clusterWizardTotalSteps,
			Title: "创建商户记录", Status: "running",
		})

		id, err := createMerchantForCluster(req)
		if err != nil {
			progress(model.ClusterWizardStepResp{
				Step: 1, Total: clusterWizardTotalSteps,
				Title: "创建商户记录", Status: "failed",
				Message: fmt.Sprintf("创建失败: %v", err),
			})
			return err
		}
		merchantId = id
		progress(model.ClusterWizardStepResp{
			Step: 1, Total: clusterWizardTotalSteps,
			Title: "创建商户记录", Status: "success",
			Message: fmt.Sprintf("商户 ID: %d", merchantId),
			MerchantId: merchantId,
		})
	}

	// 获取 AWS 账号
	logx.Infof("[ClusterWizard] 开始, merchantId=%d, cloudAccountId=%d, region=%s", merchantId, req.CloudAccountId, req.RegionId)
	acc, err := awscloud.ResolveAwsAccount(nil, merchantId, req.CloudAccountId)
	if err != nil {
		logx.Errorf("[ClusterWizard] 获取云账号失败: %v", err)
		return fmt.Errorf("获取云账号失败: %v", err)
	}
	logx.Infof("[ClusterWizard] 云账号获取成功")

	// ========== Step 2-4: 创建 EC2 实例（支持断点续传） ==========
	type nodeEC2 struct {
		stepNum      int
		title        string
		role         string
		amiId        string
		instanceType string
		volumeSize   int
		namePrefix   string
	}

	nodes := []nodeEC2{
		{2, "创建 DB EC2 实例", "db", req.DBAmiId, req.DBInstanceType, req.DBVolumeSizeGiB, "DB"},
		{3, "创建 MinIO EC2 实例", "minio", req.MinIOAmiId, req.MinIOInstanceType, req.MinIOVolumeSizeGiB, "MinIO"},
		{4, "创建 App EC2 实例", "app", req.AppAmiId, req.AppInstanceType, req.AppVolumeSizeGiB, "App"},
	}

	type ec2Result struct {
		instanceId string
		publicIP   string
		privateIP  string
		role       string
	}
	ec2Results := make([]ec2Result, len(nodes))

	// 恢复模式：查找已创建的 EC2 实例，避免重复创建
	existingNodes := make(map[string]*entity.Servers) // role → server
	if req.MerchantId > 0 {
		var existingServiceNodes []entity.MerchantServiceNodes
		dbs.DBAdmin.Where("merchant_id = ?", merchantId).Find(&existingServiceNodes)
		for _, sn := range existingServiceNodes {
			var srv entity.Servers
			if has, _ := dbs.DBAdmin.Where("id = ?", sn.ServerId).Get(&srv); has && srv.CloudInstanceId != "" {
				// 将 service_node role 映射回 wizard role
				wizardRole := sn.Role
				if wizardRole == entity.ServiceNodeRoleMinio {
					wizardRole = "minio"
				}
				existingNodes[wizardRole] = &srv
			}
		}
	}

	for i, n := range nodes {
		// 断点续传：如果该角色的 EC2 已存在且在运行，跳过创建
		if existing, ok := existingNodes[n.role]; ok {
			publicIP := existing.Host
			privateIP := existing.AuxiliaryIP
			if privateIP == "" {
				privateIP = publicIP
			}
			ec2Results[i] = ec2Result{
				instanceId: existing.CloudInstanceId,
				publicIP:   publicIP,
				privateIP:  privateIP,
				role:       n.role,
			}
			progress(model.ClusterWizardStepResp{
				Step: n.stepNum, Total: clusterWizardTotalSteps,
				Title: n.title, Status: "skipped",
				Message: fmt.Sprintf("已存在: %s, 公网 %s, 内网 %s", existing.CloudInstanceId, publicIP, privateIP),
				MerchantId: merchantId,
			})
			continue
		}

		progress(model.ClusterWizardStepResp{
			Step: n.stepNum, Total: clusterWizardTotalSteps,
			Title: n.title, Status: "running",
			MerchantId: merchantId,
		})

		// 确定 AMI
		amiId := n.amiId
		if amiId == "" {
			foundAMI, findErr := cloudaws.FindLatestUbuntuAMI(acc, req.RegionId)
			if findErr != nil {
				amiId = ""
			} else {
				amiId = foundAMI
			}
		}
		if amiId == "" {
			progress(model.ClusterWizardStepResp{
				Step: n.stepNum, Total: clusterWizardTotalSteps,
				Title: n.title, Status: "failed",
				Message: fmt.Sprintf("无法获取 %s 区域的 Ubuntu AMI", req.RegionId),
			})
			return fmt.Errorf("无法获取 AMI: region=%s", req.RegionId)
		}

		instanceType := n.instanceType
		if instanceType == "" {
			switch n.role {
			case "app":
				instanceType = "c6i.4xlarge"
			case "db":
				instanceType = "r6i.xlarge"
			default:
				instanceType = "t3.small"
			}
		}
		volumeSize := n.volumeSize
		if volumeSize == 0 {
			switch n.role {
			case "db":
				volumeSize = 200
			case "minio":
				volumeSize = 500
			case "app":
				volumeSize = 200
			default:
				volumeSize = 50
			}
		}
		serverName := fmt.Sprintf("%s-%s", req.MerchantName, n.namePrefix)

		createReq := model.AwsCreateEc2InstanceReq{
			MerchantId:     merchantId,
			CloudAccountId: req.CloudAccountId,
			RegionId:       req.RegionId,
			ImageId:        amiId,
			InstanceType:   instanceType,
			VolumeSizeGiB:  int32(volumeSize),
			InstanceName:   serverName,
			KeyName:        req.KeyName,
			SubnetId:       req.SubnetId,
		}

		instanceId, createErr := cloudaws.CreateEc2Instance(acc, createReq)
		if createErr != nil {
			progress(model.ClusterWizardStepResp{
				Step: n.stepNum, Total: clusterWizardTotalSteps,
				Title: n.title, Status: "failed",
				Message: fmt.Sprintf("创建失败: %v", createErr),
			})
			return createErr
		}

		// 等待运行（每 15 秒发心跳防止 SSE 超时）
		waitDone := make(chan struct{})
		go func() {
			_ = cloudaws.WaitForInstanceRunning(acc, req.RegionId, instanceId, 5*time.Minute)
			time.Sleep(20 * time.Second)
			close(waitDone)
		}()
		ticker := time.NewTicker(5 * time.Second)
		waitLoop:
		for {
			select {
			case <-waitDone:
				ticker.Stop()
				break waitLoop
			case <-ticker.C:
				progress(model.ClusterWizardStepResp{
					Step: n.stepNum, Total: clusterWizardTotalSteps,
					Title: n.title, Status: "running",
					Message: fmt.Sprintf("等待实例 %s 启动中...", instanceId),
					MerchantId: merchantId,
				})
			}
		}

		publicIP, _ := cloudaws.GetInstancePublicIP(acc, req.RegionId, instanceId)
		privateIP, _ := cloudaws.GetInstancePrivateIP(acc, req.RegionId, instanceId)
		if privateIP == "" {
			privateIP = publicIP
		}

		ec2Results[i] = ec2Result{instanceId: instanceId, publicIP: publicIP, privateIP: privateIP, role: n.role}

		// 立即注册服务器和 service_node，避免恢复时重复创建 EC2
		nowEarly := time.Now()
		earlyServer := &entity.Servers{
			ServerType:      1,
			MerchantId:      merchantId,
			Name:            fmt.Sprintf("%s-%s", req.MerchantName, n.namePrefix),
			Host:            publicIP,
			AuxiliaryIP:     privateIP,
			Port:            consts.DefaultPort,
			Username:        consts.DefaultUsername,
			AuthType:        1,
			Password:        consts.DefaultPassword,
			DeployPath:      consts.DeployPath,
			Status:          1,
			CloudType:       "aws",
			CloudInstanceId: instanceId,
			CloudRegionId:   req.RegionId,
			CloudAccountId:  req.CloudAccountId,
			CreatedAt:       nowEarly,
			UpdatedAt:       nowEarly,
		}
		if _, insertErr := dbs.DBAdmin.Insert(earlyServer); insertErr != nil {
			logx.Errorf("[ClusterWizard] 提前注册服务器失败: %v", insertErr)
		} else {
			svcRole := n.role
			if svcRole == "minio" {
				svcRole = entity.ServiceNodeRoleMinio
			}
			earlyNode := &entity.MerchantServiceNodes{
				MerchantId:   merchantId,
				Role:         svcRole,
				Host:         publicIP,
				ServerId:     earlyServer.Id,
				IsPrimary:    0,
				Status:       1,
				DeployStatus: entity.DeployStatusPending,
				CreatedAt:    nowEarly,
				UpdatedAt:    nowEarly,
			}
			if svcRole == "app" || svcRole == entity.ServiceNodeRoleAPI {
				earlyNode.IsPrimary = 1
			}
			dbs.DBAdmin.Insert(earlyNode)
			existingNodes[n.role] = earlyServer
		}

		progress(model.ClusterWizardStepResp{
			Step: n.stepNum, Total: clusterWizardTotalSteps,
			Title: n.title, Status: "success",
			Message: fmt.Sprintf("实例 %s, 公网 %s, 内网 %s", instanceId, publicIP, privateIP),
			MerchantId: merchantId,
		})
	}

	// ========== Step 5: 注册服务器（已在创建 EC2 后立即注册，此处做汇总确认） ==========
	serverIds := make(map[string]int) // role → server_id
	for _, r := range ec2Results {
		if existing, ok := existingNodes[r.role]; ok {
			serverIds[r.role] = existing.Id
		}
	}

	// 删除创建商户时自动创建的 role=all 节点
	dbs.DBAdmin.Where("merchant_id = ? AND role = ?", merchantId, entity.ServiceNodeRoleAll).Delete(&entity.MerchantServiceNodes{})

	// 更新商户 server_ip 为 App 节点公网 IP
	now := time.Now()
	appIP := ec2Results[2].publicIP
	if appIP != "" {
		dbs.DBAdmin.Where("id = ?", merchantId).Update(&entity.Merchants{ServerIP: appIP, UpdatedAt: now})
	}

	progress(model.ClusterWizardStepResp{
		Step: 5, Total: clusterWizardTotalSteps,
		Title: "注册服务器", Status: "success",
		Message: fmt.Sprintf("DB: %s, MinIO: %s, App: %s", ec2Results[0].publicIP, ec2Results[1].publicIP, ec2Results[2].publicIP),
		MerchantId: merchantId,
	})

	// ========== Step 6-8: 部署节点 ==========
	dbPrivateIP := ec2Results[0].privateIP
	minioPrivateIP := ec2Results[1].privateIP

	// 判断是否为 AMI 部署（服务已预装，跳过安装步骤）
	isAMIDeploy := req.DBAmiId != "" && req.MinIOAmiId != "" && req.AppAmiId != ""

	if isAMIDeploy {
		// ===== AMI 部署模式：干净 AMI，Docker 已禁用，走 DeployNode 完整部署 =====

		amiDeployNodes := []struct {
			stepNum   int
			title     string
			role      string
			serverId  int
			dbHost    string
			minioHost string
		}{
			{6, "部署 DB 节点", "db", serverIds["db"], "", ""},
			{7, "部署 MinIO 节点", "minio", serverIds["minio"], "", ""},
			{8, "部署 App 节点", "app", serverIds["app"], dbPrivateIP, minioPrivateIP},
		}

		for _, dn := range amiDeployNodes {
			progress(model.ClusterWizardStepResp{
				Step: dn.stepNum, Total: clusterWizardTotalSteps,
				Title: dn.title, Status: "running",
				MerchantId: merchantId,
			})

			if dn.serverId == 0 {
				progress(model.ClusterWizardStepResp{
					Step: dn.stepNum, Total: clusterWizardTotalSteps,
					Title: dn.title, Status: "failed",
					Message: "服务器未注册",
				})
				continue
			}

			deployReq := model.DeployNodeReq{
				ServerId:   dn.serverId,
				MerchantId: merchantId,
				NodeRole:   dn.role,
				DBHost:     dn.dbHost,
				MinioHost:  dn.minioHost,
			}

			// 异步部署 + 心跳防止 SSE 超时
			type deployResult struct {
				resp model.DeployTSDDResp
				err  error
			}
			deployDone := make(chan deployResult, 1)
			go func() {
				r, e := DeployNodeByServerId(deployReq, operator)
				deployDone <- deployResult{resp: r, err: e}
			}()

			deployTicker := time.NewTicker(5 * time.Second)
			var dResult deployResult
		amiDeployWait:
			for {
				select {
				case dResult = <-deployDone:
					deployTicker.Stop()
					break amiDeployWait
				case <-deployTicker.C:
					progress(model.ClusterWizardStepResp{
						Step: dn.stepNum, Total: clusterWizardTotalSteps,
						Title: dn.title, Status: "running",
						Message: fmt.Sprintf("正在部署 %s 节点...", dn.role),
						MerchantId: merchantId,
					})
				}
			}

			nodeId := findNodeId(merchantId, dn.serverId)
			if dResult.err != nil || !dResult.resp.Success {
				errMsg := ""
				if dResult.err != nil {
					errMsg = dResult.err.Error()
				} else {
					errMsg = dResult.resp.Message
				}
				updateNodeDeployStatus(nodeId, entity.DeployStatusFailed, errMsg, "")
				progress(model.ClusterWizardStepResp{
					Step: dn.stepNum, Total: clusterWizardTotalSteps,
					Title: dn.title, Status: "failed",
					Message: errMsg,
					MerchantId: merchantId,
				})
			} else {
				updateNodeDeployStatus(nodeId, entity.DeployStatusSuccess, "", "")
				progress(model.ClusterWizardStepResp{
					Step: dn.stepNum, Total: clusterWizardTotalSteps,
					Title: dn.title, Status: "success",
					Message: dResult.resp.Message,
					MerchantId: merchantId,
				})
			}
		}
	} else {
		// 全新部署：SSH 安装所有服务
		deployNodes := []struct {
			stepNum   int
			title     string
			role      string
			serverId  int
			dbHost    string
			minioHost string
		}{
			{6, "部署 DB 节点", "db", serverIds["db"], "", ""},
			{7, "部署 MinIO 节点", "minio", serverIds["minio"], "", ""},
			{8, "部署 App 节点", "app", serverIds["app"], dbPrivateIP, minioPrivateIP},
		}

		for _, dn := range deployNodes {
			progress(model.ClusterWizardStepResp{
				Step: dn.stepNum, Total: clusterWizardTotalSteps,
				Title: dn.title, Status: "running",
				MerchantId: merchantId,
			})

			if dn.serverId == 0 {
				progress(model.ClusterWizardStepResp{
					Step: dn.stepNum, Total: clusterWizardTotalSteps,
					Title: dn.title, Status: "failed",
					Message: "服务器未注册",
				})
				continue
			}

			deployReq := model.DeployNodeReq{
				ServerId:   dn.serverId,
				MerchantId: merchantId,
				NodeRole:   dn.role,
				DBHost:     dn.dbHost,
				MinioHost:  dn.minioHost,
			}

			// 异步部署 + 心跳防止 SSE 超时
			type deployResult struct {
				resp model.DeployTSDDResp
				err  error
			}
			deployDone := make(chan deployResult, 1)
			go func() {
				r, e := DeployNodeByServerId(deployReq, operator)
				deployDone <- deployResult{resp: r, err: e}
			}()

			deployTicker := time.NewTicker(5 * time.Second)
			var dResult deployResult
		deployWait:
			for {
				select {
				case dResult = <-deployDone:
					deployTicker.Stop()
					break deployWait
				case <-deployTicker.C:
					progress(model.ClusterWizardStepResp{
						Step: dn.stepNum, Total: clusterWizardTotalSteps,
						Title: dn.title, Status: "running",
						Message: fmt.Sprintf("正在部署 %s 节点...", dn.role),
						MerchantId: merchantId,
					})
				}
			}

			if dResult.err != nil || !dResult.resp.Success {
				errMsg := ""
				if dResult.err != nil {
					errMsg = dResult.err.Error()
				} else {
					errMsg = dResult.resp.Message
				}
				progress(model.ClusterWizardStepResp{
					Step: dn.stepNum, Total: clusterWizardTotalSteps,
					Title: dn.title, Status: "failed",
					Message: fmt.Sprintf("部署失败: %s", errMsg),
				})
				updateNodeDeployStatus(findNodeId(merchantId, dn.serverId), entity.DeployStatusFailed, errMsg, "")
				continue
			}

			updateNodeDeployStatus(findNodeId(merchantId, dn.serverId), entity.DeployStatusSuccess, "", "")
			progress(model.ClusterWizardStepResp{
				Step: dn.stepNum, Total: clusterWizardTotalSteps,
				Title: dn.title, Status: "success",
				Message: "部署成功",
			})
		}
	}

	// 在 App 节点安装 GOST + 多机 nginx 路径分发
	if appIP != "" {
		appServer := ec2Results[2]
		installReq := &model.InstallGostReq{
			Host:      appServer.publicIP,
			Port:      22,
			Username:  "ubuntu",
			MinIOHost: minioPrivateIP, // MinIO 在独立节点
			// IMHost 和 APIHost 留空 = 127.0.0.1（与 App 同机）
		}
		// 从已注册的服务器获取 SSH key
		if sid, ok := serverIds["app"]; ok {
			installReq.ServerId = sid
		}
		gostErr := InstallGostToExistingServer(installReq, func(msg string) {
			logx.Infof("[ClusterWizard] GOST: %s", msg)
		})
		if gostErr != nil {
			logx.Errorf("[ClusterWizard] App 节点 GOST 安装失败: %v", gostErr)
		}
	}

	// 为该商户关联的系统服务器创建 GOST 转发（按商户隔离，不碰其他商户的服务器）
	if appIP != "" {
		var merchant entity.Merchants
		if has, _ := dbs.DBAdmin.Where("id = ?", merchantId).Get(&merchant); has && merchant.Port > 0 {
			// 查该商户关联的 GOST 服务器（不是所有系统服务器！）
			var relations []entity.MerchantGostServers
			if err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Find(&relations); err == nil && len(relations) > 0 {
				srvIds := make([]int, 0, len(relations))
				for _, r := range relations {
					srvIds = append(srvIds, r.ServerId)
				}
				var sysServers []entity.Servers
				if err := dbs.DBAdmin.In("id", srvIds).Where("server_type = 2 AND status = 1").Find(&sysServers); err == nil {
					for _, s := range sysServers {
						tlsEnabled := s.TlsEnabled == 1
						if s.ForwardType == entity.ForwardTypeDirect {
							if tlsEnabled {
								gostapi.EnqueueCreateMerchantDirectForwardsWithTls(s.Host, merchant.Port, appIP, merchant.TunnelIP)
							} else {
								gostapi.EnqueueCreateMerchantDirectForwards(s.Host, merchant.Port, appIP, merchant.TunnelIP)
							}
						} else {
							if tlsEnabled {
								gostapi.EnqueueCreateMerchantForwardsWithTls(s.Host, merchant.Port, appIP, merchant.TunnelIP)
							} else {
								gostapi.EnqueueCreateMerchantForwards(s.Host, merchant.Port, appIP, merchant.TunnelIP)
							}
						}
					}
					logx.Infof("[ClusterWizard] 已触发 GOST 转发创建: merchant=%d, ip=%s, port=%d, servers=%d",
						merchantId, appIP, merchant.Port, len(sysServers))
				}
			} else {
				logx.Infof("[ClusterWizard] 商户 %d 未关联 GOST 服务器，跳过转发创建", merchantId)
			}
		}
	}

	return nil
}

// createMerchantForCluster 为集群向导内联创建商户（避免 deploy→merchant 循环依赖）
func createMerchantForCluster(req model.ClusterWizardReq) (int, error) {
	now := time.Now()

	expiredAt := now.AddDate(0, 0, 30)
	if req.ExpiredAt != "" {
		if t, err := time.ParseInLocation(time.DateTime, req.ExpiredAt, time.Local); err == nil {
			expiredAt = t
		}
	}

	merchant := &entity.Merchants{
		ServerIP: "pending-cluster",
		Port:     80,
		Name:     strings.TrimSpace(req.MerchantName),
		AppName:  strings.TrimSpace(req.AppName),
		Status:   1,
		PackageConfiguration: &entity.PackageConfiguration{
			DauLimit:        100,
			RegisterLimit:   100,
			GroupMemberLimit: 100,
			ExpiredAt:       expiredAt.Unix(),
		},
		ExpiredAt: expiredAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := dbs.DBAdmin.WithTx(func(session *xorm.Session) error {
		if _, err := session.Insert(merchant); err != nil {
			return err
		}
		// Insert 后根据自增 ID 生成企业号并回写
		merchant.No = utilSvc.Id2Enterprise(merchant.Id)
		if _, err := session.Where("id = ?", merchant.Id).Cols("no").Update(merchant); err != nil {
			return fmt.Errorf("回写企业号失败: %v", err)
		}
		// 自动创建 AWS 云账号
		if req.CloudAccountId > 0 {
			account := entity.CloudAccounts{
				AccountType:     "merchant",
				MerchantId:      merchant.Id,
				Name:            fmt.Sprintf("%s-aws", merchant.Name),
				CloudType:       "aws",
				Description:     fmt.Sprintf("集群商户:%s", merchant.Name),
				Status:          1,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			// 从系统账号复制凭证
			var sysAcc entity.CloudAccounts
			if has, _ := dbs.DBAdmin.Where("id = ?", req.CloudAccountId).Get(&sysAcc); has {
				account.AccessKeyId = sysAcc.AccessKeyId
				account.AccessKeySecret = sysAcc.AccessKeySecret
			}
			if _, err := session.Insert(&account); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return merchant.Id, nil
}

// findNodeId 查找 merchant_service_nodes 的 ID
func findNodeId(merchantId, serverId int) int {
	var node entity.MerchantServiceNodes
	has, _ := dbs.DBAdmin.Where("merchant_id = ? AND server_id = ?", merchantId, serverId).Get(&node)
	if has {
		return node.Id
	}
	return 0
}
