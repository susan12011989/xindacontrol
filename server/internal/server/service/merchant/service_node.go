package merchant

import (
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ServiceNodeReq 服务节点请求
type ServiceNodeReq struct {
	MerchantId int    `json:"merchant_id"`
	Role       string `json:"role" binding:"required"`
	Host       string `json:"host" binding:"required"`
	ServerId   int    `json:"server_id"`
	IsPrimary  int    `json:"is_primary"`
	Remark     string `json:"remark"`
}

// ServiceNodeResp 服务节点响应
type ServiceNodeResp struct {
	Id           int    `json:"id"`
	MerchantId   int    `json:"merchant_id"`
	Role         string `json:"role"`
	Host         string `json:"host"`
	ServerId     int    `json:"server_id"`
	IsPrimary    int    `json:"is_primary"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
	DeployStatus string `json:"deploy_status"`
	DeployError  string `json:"deploy_error"`
	LastDeployAt string `json:"last_deploy_at"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// validRoles 允许的角色
var validRoles = map[string]bool{
	entity.ServiceNodeRoleAll:   true,
	entity.ServiceNodeRoleIM:    true,
	entity.ServiceNodeRoleAPI:   true,
	entity.ServiceNodeRoleMinio: true,
	entity.ServiceNodeRoleWeb:   true,
}

// ListServiceNodes 获取商户的服务节点列表
func ListServiceNodes(merchantId int) ([]*ServiceNodeResp, error) {
	var nodes []entity.MerchantServiceNodes
	err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).
		OrderBy("is_primary DESC, role ASC").
		Find(&nodes)
	if err != nil {
		return nil, err
	}

	list := make([]*ServiceNodeResp, len(nodes))
	for i, n := range nodes {
		lastDeploy := ""
		if n.LastDeployAt != nil {
			lastDeploy = n.LastDeployAt.Format(time.DateTime)
		}
		list[i] = &ServiceNodeResp{
			Id:           n.Id,
			MerchantId:   n.MerchantId,
			Role:         n.Role,
			Host:         n.Host,
			ServerId:     n.ServerId,
			IsPrimary:    n.IsPrimary,
			Status:       n.Status,
			Remark:       n.Remark,
			DeployStatus: n.DeployStatus,
			DeployError:  n.DeployError,
			LastDeployAt: lastDeploy,
			CreatedAt:    n.CreatedAt.Format(time.DateTime),
			UpdatedAt:    n.UpdatedAt.Format(time.DateTime),
		}
	}
	return list, nil
}

// CreateServiceNode 创建服务节点
func CreateServiceNode(req ServiceNodeReq) (int, error) {
	role := strings.TrimSpace(req.Role)
	if !validRoles[role] {
		return 0, fmt.Errorf("无效的服务角色: %s，允许: all, im, api, minio, web", role)
	}
	host := strings.TrimSpace(req.Host)
	if host == "" {
		return 0, fmt.Errorf("host 不能为空")
	}

	// role=all 与其他角色互斥
	var existing []entity.MerchantServiceNodes
	err := dbs.DBAdmin.Where("merchant_id = ?", req.MerchantId).Find(&existing)
	if err != nil {
		return 0, err
	}
	for _, e := range existing {
		if e.Role == entity.ServiceNodeRoleAll && role != entity.ServiceNodeRoleAll {
			return 0, fmt.Errorf("商户当前为单机模式(role=all)，请先删除后再添加分角色节点")
		}
		if role == entity.ServiceNodeRoleAll && e.Role != entity.ServiceNodeRoleAll {
			return 0, fmt.Errorf("商户已有分角色节点，不能设为单机模式(role=all)")
		}
	}

	now := time.Now()
	node := &entity.MerchantServiceNodes{
		MerchantId: req.MerchantId,
		Role:       role,
		Host:       host,
		ServerId:   req.ServerId,
		IsPrimary:  req.IsPrimary,
		Status:     1,
		Remark:     req.Remark,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = dbs.DBAdmin.Insert(node)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return 0, fmt.Errorf("商户已存在角色为 %s 的节点", role)
		}
		return 0, err
	}

	// 如果是多机模式，同步更新商户的 nginx 配置
	go syncMerchantNginxIfNeeded(req.MerchantId)

	return node.Id, nil
}

// UpdateServiceNode 更新服务节点
func UpdateServiceNode(id int, req ServiceNodeReq) error {
	var node entity.MerchantServiceNodes
	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&node)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("节点不存在: %d", id)
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Host != "" {
		updates["host"] = strings.TrimSpace(req.Host)
	}
	if req.ServerId > 0 {
		updates["server_id"] = req.ServerId
	}
	if req.IsPrimary >= 0 {
		updates["is_primary"] = req.IsPrimary
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	_, err = dbs.DBAdmin.Table("merchant_service_nodes").Where("id = ?", id).Update(updates)
	if err != nil {
		return err
	}

	// 同步 nginx 配置
	go syncMerchantNginxIfNeeded(node.MerchantId)

	return nil
}

// DeleteServiceNode 删除服务节点
func DeleteServiceNode(id int) error {
	var node entity.MerchantServiceNodes
	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&node)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("节点不存在: %d", id)
	}

	_, err = dbs.DBAdmin.Where("id = ?", id).Delete(&entity.MerchantServiceNodes{})
	if err != nil {
		return err
	}

	go syncMerchantNginxIfNeeded(node.MerchantId)
	return nil
}

// ========== 核心查询函数 ==========

// GetMerchantServiceHosts 获取商户各服务的地址
// 返回 map[role]host，单机模式下所有角色都指向同一个 host
type MerchantServiceHosts struct {
	IMHost    string // WuKongIM 地址
	APIHost   string // tsdd-server 地址
	MinIOHost string // MinIO 地址
	WebHost   string // tsdd-web 地址
	IsCluster bool   // 是否多机模式
}

// GetMerchantServiceHosts 获取商户的服务地址映射
// 优先从 merchant_service_nodes 查询；如果没有记录，回退到 merchants.server_ip
func GetMerchantServiceHosts(merchantId int) (*MerchantServiceHosts, error) {
	var nodes []entity.MerchantServiceNodes
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Find(&nodes)
	if err != nil {
		return nil, err
	}

	// 没有节点记录，回退到 merchants.server_ip
	if len(nodes) == 0 {
		var m entity.Merchants
		has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&m)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, fmt.Errorf("商户不存在: %d", merchantId)
		}
		if m.ServerIP == "" {
			return nil, fmt.Errorf("商户服务器IP为空")
		}
		return &MerchantServiceHosts{
			IMHost:    m.ServerIP,
			APIHost:   m.ServerIP,
			MinIOHost: m.ServerIP,
			WebHost:   m.ServerIP,
			IsCluster: false,
		}, nil
	}

	// 构建角色→地址映射
	roleMap := make(map[string]string)
	for _, n := range nodes {
		roleMap[n.Role] = n.Host
	}

	// 单机模式: role=all
	if host, ok := roleMap[entity.ServiceNodeRoleAll]; ok {
		return &MerchantServiceHosts{
			IMHost:    host,
			APIHost:   host,
			MinIOHost: host,
			WebHost:   host,
			IsCluster: false,
		}, nil
	}

	// 多机模式: 各角色独立
	hosts := &MerchantServiceHosts{
		IsCluster: true,
	}
	hosts.IMHost = roleMap[entity.ServiceNodeRoleIM]
	hosts.APIHost = roleMap[entity.ServiceNodeRoleAPI]
	hosts.MinIOHost = roleMap[entity.ServiceNodeRoleMinio]
	hosts.WebHost = roleMap[entity.ServiceNodeRoleWeb]

	// 没有配的角色回退到 API 节点（通常 API 节点是主节点）
	fallback := hosts.APIHost
	if fallback == "" {
		fallback = hosts.IMHost
	}
	if hosts.IMHost == "" {
		hosts.IMHost = fallback
	}
	if hosts.APIHost == "" {
		hosts.APIHost = fallback
	}
	if hosts.MinIOHost == "" {
		hosts.MinIOHost = fallback
	}
	if hosts.WebHost == "" {
		hosts.WebHost = fallback
	}

	return hosts, nil
}

// GetMerchantPrimaryHost 获取商户主节点地址（用于 GOST 转发、API 调用等）
// 优先级：is_primary=1 → role=all → role=api → 任意节点 → merchants.server_ip
func GetMerchantPrimaryHost(merchantId int) (string, error) {
	var nodes []entity.MerchantServiceNodes
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).
		OrderBy("is_primary DESC").Find(&nodes)
	if err != nil {
		return "", err
	}

	if len(nodes) > 0 {
		// 优先 is_primary=1
		for _, n := range nodes {
			if n.IsPrimary == 1 && n.Host != "" {
				return n.Host, nil
			}
		}
		// 其次 role=all
		for _, n := range nodes {
			if n.Role == entity.ServiceNodeRoleAll && n.Host != "" {
				return n.Host, nil
			}
		}
		// 再次 role=api（多机模式下 API 节点通常是管理入口）
		for _, n := range nodes {
			if n.Role == entity.ServiceNodeRoleAPI && n.Host != "" {
				return n.Host, nil
			}
		}
		// 最后取任意有效节点
		for _, n := range nodes {
			if n.Host != "" {
				return n.Host, nil
			}
		}
	}

	// 回退到 merchants.server_ip
	var m entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&m)
	if err != nil {
		return "", err
	}
	if !has || m.ServerIP == "" {
		return "", fmt.Errorf("商户 %d 没有可用的服务器地址", merchantId)
	}
	return m.ServerIP, nil
}

// EnsureServiceNodeForMerchant 确保商户有 service_node 记录
// 用于商户创建时自动初始化
func EnsureServiceNodeForMerchant(merchantId int, serverIP string) error {
	var count int64
	count, err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).Count(&entity.MerchantServiceNodes{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有记录
	}

	now := time.Now()
	node := &entity.MerchantServiceNodes{
		MerchantId: merchantId,
		Role:       entity.ServiceNodeRoleAll,
		Host:       serverIP,
		IsPrimary:  1,
		Status:     1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err = dbs.DBAdmin.Insert(node)
	return err
}

// SwitchToClusterMode 将商户从单机模式切换到多机模式
// 删除 role=all 记录，创建指定的角色节点
func SwitchToClusterMode(merchantId int, nodes []ServiceNodeReq) error {
	// 删除旧的 all 记录
	_, err := dbs.DBAdmin.Where("merchant_id = ? AND role = ?", merchantId, entity.ServiceNodeRoleAll).
		Delete(&entity.MerchantServiceNodes{})
	if err != nil {
		return fmt.Errorf("删除单机节点失败: %v", err)
	}

	// 创建新的角色节点
	now := time.Now()
	for i, req := range nodes {
		role := strings.TrimSpace(req.Role)
		if !validRoles[role] || role == entity.ServiceNodeRoleAll {
			return fmt.Errorf("节点[%d]角色无效: %s", i, role)
		}
		host := strings.TrimSpace(req.Host)
		if host == "" {
			return fmt.Errorf("节点[%d] host 不能为空", i)
		}

		node := &entity.MerchantServiceNodes{
			MerchantId: merchantId,
			Role:       role,
			Host:       host,
			ServerId:   req.ServerId,
			IsPrimary:  req.IsPrimary,
			Status:     1,
			Remark:     req.Remark,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if _, err := dbs.DBAdmin.Insert(node); err != nil {
			return fmt.Errorf("创建节点[%d]失败: %v", i, err)
		}
	}

	// 同步 nginx 配置
	go syncMerchantNginxIfNeeded(merchantId)

	return nil
}

// syncMerchantNginxIfNeeded 如果是多机模式，重新生成并推送 nginx 配置
func syncMerchantNginxIfNeeded(merchantId int) {
	hosts, err := GetMerchantServiceHosts(merchantId)
	if err != nil {
		logx.Errorf("获取商户 %d 服务地址失败: %v", merchantId, err)
		return
	}
	if !hosts.IsCluster {
		return // 单机模式不需要特殊 nginx 配置
	}

	// TODO: 通过 SSH 推送新的 nginx 配置到商户主节点
	logx.Infof("商户 %d 多机模式 nginx 配置需要更新: IM=%s, API=%s, MinIO=%s",
		merchantId, hosts.IMHost, hosts.APIHost, hosts.MinIOHost)
}
