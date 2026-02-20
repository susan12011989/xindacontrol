package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// DeployNodeByServerId 通过已注册的服务器 ID 部署集群节点
func DeployNodeByServerId(req model.DeployNodeReq, operator string) (model.DeployTSDDResp, error) {
	var resp model.DeployTSDDResp

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
	config.MinioHost = req.DBHost // MinIO 默认与 MySQL 同机
	config.WKNodeId = req.WKNodeId
	config.WKSeedNode = req.WKSeedNode
	config.ControlAPIUsername = "merchant_api"
	config.ControlAPIPassword = "MerchantAPI@2026"

	// 设置合信达的实际端口
	config.APIPort = 5003
	config.WSPort = 5200
	config.WebPort = 82
	config.ManagerPort = 8084

	logx.Infof("[DeployNode] 开始部署 %s 节点: serverId=%d, host=%s, role=%s, wkNodeId=%d",
		req.NodeRole, req.ServerId, server.Host, req.NodeRole, req.WKNodeId)

	// 执行部署
	resp = DeployNode(client.SSHClient, config, req.ForceReset)
	resp.ServerId = req.ServerId

	if resp.Success {
		// 更新服务器信息
		dbs.DBAdmin.Where("id = ?", req.ServerId).Cols("port", "updated_at").Update(&entity.Servers{
			Port:      config.APIPort,
			UpdatedAt: time.Now(),
		})
	}

	// 记录部署历史
	logDeployHistory(req.ServerId, fmt.Sprintf("deploy_node_%s", req.NodeRole), operator, resp)

	return resp, nil
}
