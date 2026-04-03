package deploy

import (
	"fmt"
	"server/internal/server/cfg"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// DeployNodeByServerId 通过已注册的服务器 ID 部署集群节点
func DeployNodeByServerId(req model.DeployNodeReq, operator string) (model.DeployTSDDResp, error) {
	var resp model.DeployTSDDResp

	// 校验角色
	validRoles := map[string]bool{"allinone": true, "db": true, "minio": true, "app": true}
	if !validRoles[req.NodeRole] {
		return resp, fmt.Errorf("无效的节点角色: %s，允许: allinone, db, minio, app", req.NodeRole)
	}

	// app 节点必须指定 DB 地址和 MinIO 地址
	if req.NodeRole == "app" {
		if req.DBHost == "" {
			return resp, fmt.Errorf("app 节点必须指定 db_host（DB 节点内网 IP）")
		}
		if req.MinioHost == "" {
			return resp, fmt.Errorf("app 节点必须指定 minio_host（MinIO 节点内网 IP）")
		}
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
	config.MinioHost = req.MinioHost
	config.WKNodeId = req.WKNodeId
	config.WKSeedNode = req.WKSeedNode

	// app 节点的 PRIVATE_IP 应该是自身内网 IP，不是 DB 的
	// ExternalIP 已设为 server.Host（外网），AuxiliaryIP 如果有则是内网
	if server.AuxiliaryIP != "" {
		config.AppNodeIP = server.AuxiliaryIP
	} else {
		config.AppNodeIP = server.Host
	}

	// Control API 凭证从配置读取
	if cfg.C.MerchantAPI != nil {
		config.ControlAPIUsername = cfg.C.MerchantAPI.Username
		config.ControlAPIPassword = cfg.C.MerchantAPI.Password
	}

	// 使用 V2 架构端口（与 gostapi 常量一致）
	config.APIPort = gostapi.MerchantAppPortHTTP  // 5002
	config.WSPort = gostapi.MerchantAppPortWS     // 5200
	config.WebPort = 82
	config.ManagerPort = 8084

	logx.Infof("[DeployNode] 开始部署 %s 节点: serverId=%d, host=%s, role=%s, wkNodeId=%d",
		req.NodeRole, req.ServerId, server.Host, req.NodeRole, req.WKNodeId)

	// 执行部署
	resp = DeployNode(client.SSHClient, config, req.ForceReset)
	resp.ServerId = req.ServerId

	if resp.Success {
		// 更新部署时间（不更新 port，port 始终为 SSH 端口）
		dbs.DBAdmin.Where("id = ?", req.ServerId).Cols("updated_at").Update(&entity.Servers{
			UpdatedAt: time.Now(),
		})
	}

	// 记录部署历史
	logDeployHistory(req.ServerId, fmt.Sprintf("deploy_node_%s", req.NodeRole), operator, resp)

	return resp, nil
}

// DeployNodeTracked 带状态跟踪的节点部署
// 自动创建/查找 service_node 记录，更新部署状态
func DeployNodeTracked(req model.DeployNodeReq, operator string) (model.DeployTSDDResp, error) {
	// 确保 merchant_service_nodes 记录存在
	nodeId, err := ensureServiceNode(req)
	if err != nil {
		return model.DeployTSDDResp{}, fmt.Errorf("创建服务节点记录失败: %v", err)
	}

	// 检查是否正在部署
	var node entity.MerchantServiceNodes
	dbs.DBAdmin.Where("id = ?", nodeId).Get(&node)
	if node.DeployStatus == entity.DeployStatusDeploying {
		return model.DeployTSDDResp{}, fmt.Errorf("该节点正在部署中，请稍后再试")
	}

	// 标记为部署中
	updateNodeDeployStatus(nodeId, entity.DeployStatusDeploying, "", "")

	// 执行部署
	resp, err := DeployNodeByServerId(req, operator)

	// 更新部署状态
	if err != nil {
		updateNodeDeployStatus(nodeId, entity.DeployStatusFailed, err.Error(), "")
		return resp, err
	}

	if resp.Success {
		// 提取输出摘要（最后几个步骤）
		output := ""
		for _, step := range resp.Steps {
			if step.Status == "failed" {
				output += fmt.Sprintf("[%s] %s: %s\n", step.Status, step.Name, step.Message)
			}
		}
		updateNodeDeployStatus(nodeId, entity.DeployStatusSuccess, "", output)
	} else {
		errMsg := resp.Message
		if errMsg == "" {
			for _, step := range resp.Steps {
				if step.Status == "failed" {
					errMsg = fmt.Sprintf("%s: %s", step.Name, step.Message)
					break
				}
			}
		}
		updateNodeDeployStatus(nodeId, entity.DeployStatusFailed, errMsg, "")
	}

	return resp, nil
}

// RetryDeploy 重试失败的部署
func RetryDeploy(nodeId int, operator string) (model.DeployTSDDResp, error) {
	var node entity.MerchantServiceNodes
	has, err := dbs.DBAdmin.Where("id = ?", nodeId).Get(&node)
	if err != nil {
		return model.DeployTSDDResp{}, err
	}
	if !has {
		return model.DeployTSDDResp{}, fmt.Errorf("节点不存在: %d", nodeId)
	}
	if node.DeployStatus != entity.DeployStatusFailed && node.DeployStatus != "" {
		return model.DeployTSDDResp{}, fmt.Errorf("只能重试失败的部署，当前状态: %s", node.DeployStatus)
	}
	if node.ServerId == 0 {
		return model.DeployTSDDResp{}, fmt.Errorf("节点未关联服务器，无法部署")
	}

	// 构造部署请求（从节点记录恢复 DB/MinIO 地址）
	req := model.DeployNodeReq{
		ServerId:   node.ServerId,
		MerchantId: node.MerchantId,
		NodeRole:   node.Role,
		DBHost:     node.DbHost,
		MinioHost:  node.MinioHost,
		WKNodeId:   node.WkNodeId,
		ForceReset: false,
	}

	return DeployNodeTracked(req, operator)
}

// GetClusterTopology 获取商户集群拓扑
func GetClusterTopology(merchantId int) (*model.ClusterTopologyResp, error) {
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("商户不存在: %d", merchantId)
	}

	var nodes []entity.MerchantServiceNodes
	err = dbs.DBAdmin.Where("merchant_id = ?", merchantId).OrderBy("is_primary DESC, role ASC").Find(&nodes)
	if err != nil {
		return nil, err
	}

	// 批量查询关联的服务器信息
	serverIds := make([]int, 0)
	for _, n := range nodes {
		if n.ServerId > 0 {
			serverIds = append(serverIds, n.ServerId)
		}
	}
	serverMap := make(map[int]*entity.Servers)
	if len(serverIds) > 0 {
		var servers []entity.Servers
		dbs.DBAdmin.In("id", serverIds).Find(&servers)
		for i := range servers {
			serverMap[servers[i].Id] = &servers[i]
		}
	}

	// 判断部署模式
	mode := "single"
	for _, n := range nodes {
		if n.Role != entity.ServiceNodeRoleAll {
			mode = "cluster"
			break
		}
	}

	resp := &model.ClusterTopologyResp{
		MerchantId:   merchant.Id,
		MerchantName: merchant.Name,
		DeployMode:   mode,
		Nodes:        make([]model.ClusterNodeInfo, len(nodes)),
	}

	for i, n := range nodes {
		lastDeploy := ""
		if n.LastDeployAt != nil {
			lastDeploy = n.LastDeployAt.Format(time.DateTime)
		}
		serverName := ""
		privateIP := n.Host // 默认回退到 host
		if s, ok := serverMap[n.ServerId]; ok {
			serverName = s.Name
			if s.AuxiliaryIP != "" {
				privateIP = s.AuxiliaryIP
			}
		}
		resp.Nodes[i] = model.ClusterNodeInfo{
			NodeId:       n.Id,
			MerchantId:   n.MerchantId,
			Role:         n.Role,
			Host:         n.Host,
			PrivateIP:    privateIP,
			ServerId:     n.ServerId,
			ServerName:   serverName,
			IsPrimary:    n.IsPrimary,
			Status:       n.Status,
			WKNodeId:     n.WkNodeId,
			DBHost:       n.DbHost,
			MinioHost:    n.MinioHost,
			DeployStatus: n.DeployStatus,
			DeployError:  n.DeployError,
			LastDeployAt: lastDeploy,
		}
	}

	return resp, nil
}

// ensureServiceNode 确保 merchant_service_nodes 记录存在，返回 node ID
func ensureServiceNode(req model.DeployNodeReq) (int, error) {
	role := req.NodeRole
	if role == "allinone" {
		role = entity.ServiceNodeRoleAll
	}

	// 查找已有记录
	var node entity.MerchantServiceNodes
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND server_id = ?", req.MerchantId, req.ServerId).Get(&node)
	if err != nil {
		return 0, err
	}
	if has {
		// 更新角色和部署参数
		updates := map[string]interface{}{
			"role":       role,
			"updated_at": time.Now(),
		}
		if req.WKNodeId > 0 {
			updates["wk_node_id"] = req.WKNodeId
		}
		if req.DBHost != "" {
			updates["db_host"] = req.DBHost
		}
		if req.MinioHost != "" {
			updates["minio_host"] = req.MinioHost
		}
		dbs.DBAdmin.Where("id = ?", node.Id).Update(updates)
		return node.Id, nil
	}

	// 获取服务器 host
	var server entity.Servers
	has, _ = dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	host := ""
	if has {
		host = server.Host
	}

	now := time.Now()
	newNode := &entity.MerchantServiceNodes{
		MerchantId:   req.MerchantId,
		Role:         role,
		Host:         host,
		ServerId:     req.ServerId,
		IsPrimary:    0,
		Status:       1,
		WkNodeId:     req.WKNodeId,
		DbHost:       req.DBHost,
		MinioHost:    req.MinioHost,
		DeployStatus: entity.DeployStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	// app 节点默认为主节点
	if role == entity.ServiceNodeRoleAPI || role == "app" {
		newNode.IsPrimary = 1
	}

	_, err = dbs.DBAdmin.Insert(newNode)
	if err != nil {
		return 0, err
	}
	return newNode.Id, nil
}

// updateNodeDeployStatus 更新节点部署状态
func updateNodeDeployStatus(nodeId int, status, errMsg, output string) {
	now := time.Now()
	updates := map[string]interface{}{
		"deploy_status":  status,
		"deploy_error":   errMsg,
		"deploy_output":  output,
		"last_deploy_at": &now,
		"updated_at":     now,
	}
	dbs.DBAdmin.Table("merchant_service_nodes").Where("id = ?", nodeId).Update(updates)
}
