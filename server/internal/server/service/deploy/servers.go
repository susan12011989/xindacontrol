package deploy

import (
	"errors"
	"fmt"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)


// QueryServers 查询服务器列表
func QueryServers(req model.QueryServersReq) (model.QueryServersResponse, error) {
	resp := model.QueryServersResponse{
		List: []model.ServerResp{}, // 初始化为空切片，避免 JSON 序列化为 null
	}

	session := dbs.DBAdmin.Table("servers")

	if req.Name != "" {
		session = session.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Host != "" {
		session = session.Where("host LIKE ?", "%"+req.Host+"%")
	}
	if req.Status != nil {
		session = session.Where("status = ?", *req.Status)
	}
	if req.ServerType != nil {
		session = session.Where("server_type = ?", *req.ServerType)
	}
	if req.MerchantId != nil && *req.MerchantId > 0 {
		session = session.Where("merchant_id = ?", *req.MerchantId)
	}
	if req.GroupId != nil {
		session = session.Where("group_id = ?", *req.GroupId)
	}

	offset := (req.Page - 1) * req.Size
	var servers []entity.Servers
	total, err := session.Desc("id").
		Limit(req.Size, offset).
		FindAndCount(&servers)
	if err != nil {
		logx.Errorf("query servers err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 收集所有商户ID，批量查询商户信息
	merchantIds := make([]int, 0)
	for _, s := range servers {
		if s.MerchantId > 0 {
			merchantIds = append(merchantIds, s.MerchantId)
		}
	}

	// 批量获取商户信息
	merchantMap := make(map[int]*entity.Merchants)
	if len(merchantIds) > 0 {
		var merchants []entity.Merchants
		if err := dbs.DBAdmin.In("id", merchantIds).Find(&merchants); err != nil {
			logx.Errorf("query merchants for servers err: %+v", err)
		} else {
			for i := range merchants {
				merchantMap[merchants[i].Id] = &merchants[i]
			}
		}
	}

	// 收集所有分组ID，批量查询分组信息
	groupIds := make([]int, 0)
	for _, s := range servers {
		if s.GroupId > 0 {
			groupIds = append(groupIds, s.GroupId)
		}
	}
	groupMap := make(map[int]string)
	if len(groupIds) > 0 {
		var groups []entity.ResourceGroups
		if err := dbs.DBAdmin.In("id", groupIds).Find(&groups); err == nil {
			for _, g := range groups {
				groupMap[g.Id] = g.Name
			}
		}
	}

	for _, s := range servers {
		item := model.ServerResp{
			Id:          s.Id,
			Name:        s.Name,
			Host:        s.Host,
			AuxiliaryIP: s.AuxiliaryIP,
			Port:        s.Port,
			Username:    s.Username,
			AuthType:    s.AuthType,
			ServerType:  s.ServerType,
			ForwardType: s.ForwardType,
			Status:      s.Status,
			TlsEnabled:  s.TlsEnabled,
			Description: s.Description,
			MerchantId:  s.MerchantId,
			GroupId:     s.GroupId,
			CreatedAt:   s.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   s.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if s.TlsDeployedAt != nil {
			item.TlsDeployedAt = s.TlsDeployedAt.Format("2006-01-02 15:04:05")
		}
		// 填充商户信息
		if merchant, ok := merchantMap[s.MerchantId]; ok {
			item.MerchantName = merchant.Name
			item.MerchantNo = merchant.No
		}
		// 填充分组信息
		if name, ok := groupMap[s.GroupId]; ok {
			item.GroupName = name
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// GetServerDetail 获取服务器详情
func GetServerDetail(id int) (model.ServerResp, error) {
	var resp model.ServerResp

	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&server)
	if err != nil {
		logx.Errorf("get server err: %+v", err)
		return resp, err
	}
	if !has {
		return resp, errors.New("服务器不存在")
	}

	resp = model.ServerResp{
		Id:          server.Id,
		Name:        server.Name,
		Host:        server.Host,
		AuxiliaryIP: server.AuxiliaryIP,
		Port:        server.Port,
		Username:    server.Username,
		AuthType:    server.AuthType,
		ServerType:  server.ServerType,
		ForwardType: server.ForwardType,
		Status:      server.Status,
		TlsEnabled:  server.TlsEnabled,
		Description: server.Description,
		MerchantId:  server.MerchantId,
		GroupId:     server.GroupId,
		CreatedAt:   server.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   server.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if server.TlsDeployedAt != nil {
		resp.TlsDeployedAt = server.TlsDeployedAt.Format("2006-01-02 15:04:05")
	}

	// 获取商户信息
	if server.MerchantId > 0 {
		var merchant entity.Merchants
		if has, err := dbs.DBAdmin.Where("id = ?", server.MerchantId).Get(&merchant); err == nil && has {
			resp.MerchantName = merchant.Name
			resp.MerchantNo = merchant.No
		}
	}

	// 获取分组信息
	if server.GroupId > 0 {
		var group entity.ResourceGroups
		if has, err := dbs.DBAdmin.Where("id = ?", server.GroupId).Get(&group); err == nil && has {
			resp.GroupName = group.Name
		}
	}

	return resp, nil
}

// CreateServer 创建服务器
func CreateServer(req model.CreateServerReq) (int, error) {
	// 检查同名服务器
	has, err := dbs.DBAdmin.Where("name = ?", req.Name).Exist(&entity.Servers{})
	if err != nil {
		logx.Errorf("check server name err: %+v", err)
		return 0, err
	}
	if has {
		return 0, errors.New("已存在同名服务器")
	}

	now := time.Now()
	serverType := req.ServerType
	if serverType == 0 {
		serverType = 1 // 默认为商户服务器
	}
	forwardType := req.ForwardType
	if forwardType == 0 {
		forwardType = entity.ForwardTypeEncrypted // 默认为加密转发
	}
	server := entity.Servers{
		Name:        req.Name,
		Host:        req.Host,
		AuxiliaryIP: req.AuxiliaryIP,
		Port:        req.Port,
		Username:    req.Username,
		AuthType:    req.AuthType,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		ServerType:  serverType,
		ForwardType: forwardType,
		MerchantId:  req.MerchantId,
		GroupId:     req.GroupId,
		Status:      1,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	affected, err := dbs.DBAdmin.Insert(&server)
	if err != nil {
		logx.Errorf("create server err: %+v", err)
		return 0, err
	}
	if affected == 0 {
		return 0, errors.New("创建失败")
	}

	// 系统服务器：为所有商户创建 gost 转发服务
	if serverType == 2 {
		go enqueueGostServicesForMerchants(server.Host, forwardType)
	}

	return server.Id, nil
}

// enqueueGostServicesForMerchants 为指定系统服务器入队创建所有商户的转发任务
// forwardType: 1-加密转发 2-直连转发
func enqueueGostServicesForMerchants(serverHost string, forwardType int) {
	// 查询所有有效商户
	var merchants []entity.Merchants
	if err := dbs.DBAdmin.Where("status = 1 AND expired_at > ?", time.Now()).Find(&merchants); err != nil {
		logx.Errorf("query merchants for gost service creation err: %+v", err)
		return
	}

	if len(merchants) == 0 {
		logx.Infof("no valid merchants found for gost service creation on server %s", serverHost)
		return
	}

	// 检查该服务器是否启用了 TLS
	var server entity.Servers
	tlsEnabled := false
	has, err := dbs.DBAdmin.Where("host = ? AND server_type = 2", serverHost).Get(&server)
	if err == nil && has && server.TlsEnabled == 1 {
		tlsEnabled = true
	}

	forwardTypeName := "encrypted"
	if forwardType == entity.ForwardTypeDirect {
		forwardTypeName = "direct"
	}
	if tlsEnabled {
		forwardTypeName += "+tls-listener"
	}

	for _, m := range merchants {
		var err error
		if forwardType == entity.ForwardTypeDirect {
			if tlsEnabled {
				err = gostapi.EnqueueCreateMerchantDirectForwardsWithTls(serverHost, m.Port, m.ServerIP, m.TunnelIP)
			} else {
				err = gostapi.EnqueueCreateMerchantDirectForwards(serverHost, m.Port, m.ServerIP, m.TunnelIP)
			}
		} else {
			if tlsEnabled {
				err = gostapi.EnqueueCreateMerchantForwardsWithTls(serverHost, m.Port, m.ServerIP, m.TunnelIP)
			} else {
				err = gostapi.EnqueueCreateMerchantForwards(serverHost, m.Port, m.ServerIP, m.TunnelIP)
			}
		}
		if err != nil {
			logx.Errorf("enqueue create merchant %s forwards task for merchant %d (port %d) on server %s failed: %+v",
				forwardTypeName, m.Id, m.Port, serverHost, err)
		}
	}
	logx.Infof("enqueued create %s gost services tasks for %d merchants on server %s", forwardTypeName, len(merchants), serverHost)
}

// UpdateServer 更新服务器
func UpdateServer(id int, req model.UpdateServerReq) error {
	// 先获取原服务器信息（用于判断 Host 是否变更）
	var oldServer entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&oldServer)
	if err != nil {
		logx.Errorf("get server err: %+v", err)
		return err
	}
	if !has {
		return errors.New("服务器不存在")
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Host != "" {
		updates["host"] = req.Host
	}
	if req.AuxiliaryIP != nil {
		updates["auxiliary_ip"] = *req.AuxiliaryIP
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.AuthType != nil {
		updates["auth_type"] = *req.AuthType
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}
	if req.PrivateKey != "" {
		updates["private_key"] = req.PrivateKey
	}
	if req.ServerType != nil {
		updates["server_type"] = *req.ServerType
	}
	if req.ForwardType != nil {
		updates["forward_type"] = *req.ForwardType
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.MerchantId != nil {
		updates["merchant_id"] = *req.MerchantId
	}
	if req.GroupId != nil {
		updates["group_id"] = *req.GroupId
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if len(updates) == 0 {
		return errors.New("没有需要更新的字段")
	}

	updates["updated_at"] = time.Now()

	affected, err := dbs.DBAdmin.Table("servers").
		Where("id = ?", id).
		Update(updates)
	if err != nil {
		logx.Errorf("update server err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("服务器不存在或无变更")
	}

	// 清除 SSH 连接池缓存（Host、密码、密钥等变更后需要重新连接）
	pool := utils.GetSSHPool()
	pool.RemoveConnection(fmt.Sprintf("%d", id))
	logx.Infof("cleared SSH connection cache for server %d", id)

	// 系统服务器 Host 变更时，需要在新服务器上为所有商户创建 gost 转发
	if oldServer.ServerType == 2 && req.Host != "" && req.Host != oldServer.Host {
		forwardType := oldServer.ForwardType
		if req.ForwardType != nil {
			forwardType = *req.ForwardType
		}
		go enqueueGostServicesForMerchants(req.Host, forwardType)
		logx.Infof("system server %d host changed from %s to %s, enqueued gost services for all merchants",
			id, oldServer.Host, req.Host)
	}

	return nil
}

// DeleteServer 删除服务器
func DeleteServer(id int) error {
	// 先获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&server)
	if err != nil {
		logx.Errorf("get server err: %+v", err)
		return err
	}
	if !has {
		return errors.New("服务器不存在")
	}

	// 删除服务器记录
	affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.Servers{})
	if err != nil {
		logx.Errorf("delete server err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("删除失败")
	}

	// 系统服务器：删除该服务器上所有商户的 gost 转发服务
	if server.ServerType == 2 {
		go enqueueDeleteGostServicesForMerchants(server.Host, server.ForwardType)
	}

	return nil
}

// enqueueDeleteGostServicesForMerchants 为指定系统服务器入队删除所有商户的转发任务
// forwardType: 1-加密转发 2-直连转发
func enqueueDeleteGostServicesForMerchants(serverHost string, forwardType int) {
	// 查询所有有效商户
	var merchants []entity.Merchants
	if err := dbs.DBAdmin.Where("status = 1 AND expired_at > ?", time.Now()).Find(&merchants); err != nil {
		logx.Errorf("query merchants for gost service deletion err: %+v", err)
		return
	}

	if len(merchants) == 0 {
		logx.Infof("no valid merchants found for gost service deletion on server %s", serverHost)
		return
	}

	forwardTypeName := "encrypted"
	if forwardType == entity.ForwardTypeDirect {
		forwardTypeName = "direct"
	}

	for _, m := range merchants {
		var err error
		if forwardType == entity.ForwardTypeDirect {
			// 直连转发
			err = gostapi.EnqueueDeleteMerchantDirectForwards(serverHost, m.Port)
		} else {
			// 加密转发
			err = gostapi.EnqueueDeleteMerchantForwards(serverHost, m.Port)
		}
		if err != nil {
			logx.Errorf("enqueue delete merchant %s forwards task for merchant %d (port %d) on server %s failed: %+v",
				forwardTypeName, m.Id, m.Port, serverHost, err)
		}
	}
	logx.Infof("enqueued delete %s gost services tasks for %d merchants on server %s", forwardTypeName, len(merchants), serverHost)
}

// TestConnection 测试SSH连接
func TestConnection(req model.TestConnectionReq) error {
	return utils.TestSSHConnection(req.Host, req.Port, req.Username, req.Password, req.PrivateKey)
}

// GetSSHClient 获取SSH客户端（使用连接池）
func GetSSHClient(serverId int) (*utils.PooledSSHClient, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return nil, errors.New("服务器不存在")
	}
	if server.Status != 1 {
		return nil, errors.New("服务器已禁用")
	}

	pool := utils.GetSSHPool()
	key := fmt.Sprintf("server_%d", serverId)
	client, err := pool.GetOrCreateConnection(
		key,
		server.Host,
		server.Port,
		server.Username,
		server.Password,
		server.PrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("获取SSH连接失败: %v", err)
	}

	return client, nil
}
