package deploy

import (
	"fmt"
	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	cloudaws "server/internal/server/service/cloud_aws"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// DeployNodeByServerId 通过已注册的服务器 ID 部署集群节点
// 当 ServerId == 0 且 AmiId 不为空时，先创建 EC2 实例并注册服务器
func DeployNodeByServerId(req model.DeployNodeReq, operator string) (model.DeployTSDDResp, error) {
	var resp model.DeployTSDDResp

	// 如果未指定服务器但提供了 AMI，自动创建 EC2 + 注册
	if req.ServerId == 0 && req.AmiId != "" {
		serverId, err := createAndRegisterEc2(req)
		if err != nil {
			return resp, fmt.Errorf("创建 EC2 失败: %v", err)
		}
		req.ServerId = serverId
	}

	// 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return resp, fmt.Errorf("服务器不存在")
	}

	// 获取SSH客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, fmt.Errorf("连接服务器失败: %v", err)
	}

	// 构建部署配置
	config := model.DefaultDeployConfig
	config.ExternalIP = server.Host
	config.NodeRole = req.NodeRole
	config.DBHost = req.DBHost
	config.RedisHost = req.DBHost // Redis 默认与 MySQL 同机
	config.WKNodeId = req.WKNodeId
	config.WKSeedNode = req.WKSeedNode
	config.ControlAPIUsername = "merchant_api"
	config.ControlAPIPassword = "MerchantAPI@2026"
	config.FromAMI = req.AmiId != ""

	// MinIO 地址：优先使用单独指定的，否则与 DBHost 相同
	if req.MinioHost != "" {
		config.MinioHost = req.MinioHost
	} else {
		config.MinioHost = req.DBHost
	}

	// MinIO 公网地址（presigned URL 直传用）：从集群拓扑中查找 MinIO 节点的公网 IP
	config.MinioPublicHost = lookupMinioPublicIP(req.MerchantId)

	// 设置业务端口（与 GOST 本地转发目标端口对齐）
	// APIPort=10002 对应 MerchantAppPortHTTP，GOST 转发 10012→10002
	config.APIPort = 10002
	config.WSPort = 5200
	config.WebPort = 82
	config.ManagerPort = 8084

	// 检测节点自身内网 IP（WuKongIM 集群通信用）
	if privateIP := DetectPrivateIP(client.SSHClient); privateIP != "" {
		config.PrivateIP = privateIP
		logx.Infof("[DeployNode] 检测到内网IP: %s", privateIP)
	} else {
		config.PrivateIP = server.Host // 回退到注册的 Host
		logx.Infof("[DeployNode] 未检测到内网IP，使用 Host: %s", server.Host)
	}

	logx.Infof("[DeployNode] 开始部署 %s 节点: serverId=%d, host=%s, privateIP=%s, role=%s, wkNodeId=%d",
		req.NodeRole, req.ServerId, server.Host, config.PrivateIP, req.NodeRole, req.WKNodeId)

	// 执行部署
	resp = DeployNode(client.SSHClient, config, req.ForceReset)
	resp.ServerId = req.ServerId

	if resp.Success {
		// 更新服务器信息（注意：Port 字段是 SSH 端口，不能覆盖为 API 端口）
		dbs.DBAdmin.Where("id = ?", req.ServerId).Cols("updated_at").Update(&entity.Servers{
			UpdatedAt: time.Now(),
		})

		// 保存集群拓扑信息
		saveClusterNode(req, config)

		// App 节点部署成功后，自动同步 GOST 转发
		if req.NodeRole == "app" {
			go func() {
				results, err := SyncClusterGostForward(req.MerchantId)
				if err != nil {
					logx.Errorf("[DeployNode] 自动同步 GOST 失败: %v", err)
				} else {
					for _, r := range results {
						if r.Success {
							logx.Infof("[DeployNode] GOST %s -> %s 同步成功", r.ServerHost, r.TargetIP)
						}
					}
				}
			}()
		}
	}

	// 记录部署历史
	logDeployHistory(req.ServerId, fmt.Sprintf("deploy_node_%s", req.NodeRole), operator, resp)

	return resp, nil
}

// GetClusterNodes 获取商户的集群节点拓扑
func GetClusterNodes(merchantId int) ([]model.ClusterNodeInfo, error) {
	var nodes []entity.ClusterNodes
	err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).OrderBy("node_role, id").Find(&nodes)
	if err != nil {
		return nil, fmt.Errorf("查询集群节点失败: %v", err)
	}

	result := make([]model.ClusterNodeInfo, 0, len(nodes))
	for _, n := range nodes {
		// 查关联的服务器名和公网 IP
		var server entity.Servers
		dbs.DBAdmin.Where("id = ?", n.ServerId).Cols("name", "host").Get(&server)

		result = append(result, model.ClusterNodeInfo{
			Id:         n.Id,
			MerchantId: n.MerchantId,
			ServerId:   n.ServerId,
			ServerName: server.Name,
			ServerHost: server.Host,
			NodeRole:   n.NodeRole,
			PrivateIP:  n.PrivateIP,
			WKNodeId:   n.WKNodeId,
			DBHost:     n.DBHost,
			MinioHost:  n.MinioHost,
			Status:     n.Status,
			DeployedAt: n.DeployedAt,
		})
	}

	return result, nil
}

// saveClusterNode 保存或更新集群节点拓扑
func saveClusterNode(req model.DeployNodeReq, config model.DeployConfig) {
	now := time.Now()
	node := entity.ClusterNodes{
		MerchantId: req.MerchantId,
		ServerId:   req.ServerId,
		NodeRole:   req.NodeRole,
		PrivateIP:  config.PrivateIP,
		WKNodeId:   req.WKNodeId,
		DBHost:     req.DBHost,
		MinioHost:  config.MinioHost,
		Status:     entity.ClusterStatusDeployed,
		DeployedAt: &now,
		UpdatedAt:  now,
	}

	// 同一个商户+服务器，更新已有记录；否则插入新记录
	var existing entity.ClusterNodes
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND server_id = ?", req.MerchantId, req.ServerId).Get(&existing)
	if err != nil {
		logx.Errorf("[saveClusterNode] 查询失败: %v", err)
		return
	}

	if has {
		node.Id = existing.Id
		_, err = dbs.DBAdmin.Where("id = ?", existing.Id).AllCols().Update(&node)
	} else {
		node.CreatedAt = now
		_, err = dbs.DBAdmin.Insert(&node)
	}
	if err != nil {
		logx.Errorf("[saveClusterNode] 保存失败: %v", err)
	} else {
		logx.Infof("[saveClusterNode] 已记录 %s 节点: merchantId=%d, serverId=%d, privateIP=%s",
			req.NodeRole, req.MerchantId, req.ServerId, config.PrivateIP)
	}
}

// createAndRegisterEc2 创建 EC2 实例并注册为服务器，返回 server ID
func createAndRegisterEc2(req model.DeployNodeReq) (int, error) {
	// 获取 AWS 云账号
	acc, err := awscloud.ResolveAwsAccount(nil, req.MerchantId, req.CloudAccountId)
	if err != nil {
		return 0, fmt.Errorf("获取 AWS 账号失败: %v", err)
	}

	// 获取商户名称（用于命名）
	var merchant entity.Merchants
	dbs.DBAdmin.Where("id = ?", req.MerchantId).Cols("name").Get(&merchant)
	serverName := fmt.Sprintf("%s-%s", merchant.Name, req.NodeRole)
	if merchant.Name == "" {
		serverName = fmt.Sprintf("merchant%d-%s", req.MerchantId, req.NodeRole)
	}

	// 设置默认值
	instanceType := req.InstanceType
	if instanceType == "" {
		switch req.NodeRole {
		case "db":
			instanceType = "r5.large"
		case "minio":
			instanceType = "t3.medium"
		default:
			instanceType = "t3.large"
		}
	}
	volumeSize := req.VolumeSizeGiB
	if volumeSize == 0 {
		switch req.NodeRole {
		case "db":
			volumeSize = 100
		case "minio":
			volumeSize = 200
		default:
			volumeSize = 30
		}
	}

	// 从同商户已有 EC2 继承安全组
	var securityGroupIds []string
	var existingSrv entity.Servers
	if has, _ := dbs.DBAdmin.Where("merchant_id = ? AND aws_instance_id != ''", req.MerchantId).
		Cols("aws_instance_id", "aws_region_id").Limit(1).Get(&existingSrv); has {
		sgs, sgErr := cloudaws.GetInstanceSecurityGroups(acc, existingSrv.AwsRegionId, existingSrv.AwsInstanceId)
		if sgErr == nil && len(sgs) > 0 {
			securityGroupIds = sgs
			logx.Infof("[CreateEC2] 从已有实例 %s 继承安全组: %v", existingSrv.AwsInstanceId, sgs)
		} else if sgErr != nil {
			logx.Errorf("[CreateEC2] 获取已有实例安全组失败: %v", sgErr)
		}
	}

	logx.Infof("[CreateEC2] 创建 %s EC2: AMI=%s, 机型=%s, 磁盘=%dGB", req.NodeRole, req.AmiId, instanceType, volumeSize)

	// 创建 EC2
	createReq := model.AwsCreateEc2InstanceReq{
		MerchantId:       req.MerchantId,
		RegionId:         req.RegionId,
		ImageId:          req.AmiId,
		InstanceType:     instanceType,
		SubnetId:         req.SubnetId,
		KeyName:          req.KeyName,
		VolumeSizeGiB:    volumeSize,
		InstanceName:     serverName,
		SecurityGroupIds: securityGroupIds,
	}

	instanceId, err := cloudaws.CreateEc2Instance(acc, createReq)
	if err != nil {
		return 0, fmt.Errorf("创建 EC2 实例失败: %v", err)
	}
	logx.Infof("[CreateEC2] 实例已创建: %s，等待启动...", instanceId)

	// 等待实例运行
	if err := cloudaws.WaitForInstanceRunning(acc, req.RegionId, instanceId, 5*time.Minute); err != nil {
		return 0, fmt.Errorf("等待 EC2 启动超时: %v", err)
	}

	time.Sleep(30 * time.Second) // 等待系统初始化

	// 获取公网 IP
	publicIP, err := cloudaws.GetInstancePublicIP(acc, req.RegionId, instanceId)
	if err != nil {
		return 0, fmt.Errorf("获取公网 IP 失败: %v", err)
	}
	logx.Infof("[CreateEC2] 实例就绪: %s, IP: %s", instanceId, publicIP)

	// 从同商户已有服务器继承 SSH 认证方式（密钥优先）
	authType := 1
	sshPassword := consts.DefaultPassword
	sshPrivateKey := ""
	var existingServer entity.Servers
	if has, _ := dbs.DBAdmin.Where("merchant_id = ? AND private_key != ''", req.MerchantId).
		Cols("auth_type", "username", "private_key").Limit(1).Get(&existingServer); has {
		authType = existingServer.AuthType
		sshPrivateKey = existingServer.PrivateKey
		sshPassword = ""
		logx.Infof("[CreateEC2] 从已有服务器继承 SSH 密钥认证")
	}

	// 注册服务器
	now := time.Now()
	server := &entity.Servers{
		ServerType:    1,
		MerchantId:    req.MerchantId,
		Name:          serverName,
		Host:          publicIP,
		Port:          22,
		Username:      "ubuntu",
		AuthType:      authType,
		Password:      sshPassword,
		PrivateKey:    sshPrivateKey,
		AwsInstanceId:  instanceId,
		AwsRegionId:    req.RegionId,
		CloudType:       "aws",
		CloudInstanceId: instanceId,
		CloudRegionId:   req.RegionId,
		Status:        1,
		Description:   fmt.Sprintf("集群%s节点-扩容创建", req.NodeRole),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := dbs.DBAdmin.Insert(server); err != nil {
		return 0, fmt.Errorf("注册服务器失败: %v", err)
	}
	logx.Infof("[CreateEC2] 服务器已注册: id=%d, host=%s", server.Id, publicIP)

	// 等待 SSH 就绪
	if !waitSSHReadyWithKey(publicIP, "ubuntu", sshPassword, sshPrivateKey, 3*time.Minute) {
		logx.Errorf("[CreateEC2] SSH 连接 %s 超时，继续尝试部署...", publicIP)
	}

	return server.Id, nil
}

// lookupMinioPublicIP 从集群拓扑查找 MinIO 节点的公网 IP（presigned URL 需要客户端直连）
func lookupMinioPublicIP(merchantId int) string {
	var minioNode entity.ClusterNodes
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND node_role = ?", merchantId, entity.ClusterRoleMinio).
		Cols("server_id").Limit(1).Get(&minioNode)
	if err != nil || !has {
		return "" // 未找到 MinIO 节点，回退到 ExternalIP（allinone 模式）
	}

	var server entity.Servers
	has, err = dbs.DBAdmin.Where("id = ?", minioNode.ServerId).Cols("host").Get(&server)
	if err != nil || !has {
		return ""
	}

	logx.Infof("[lookupMinioPublicIP] merchantId=%d, MinIO公网IP=%s", merchantId, server.Host)
	return server.Host
}
