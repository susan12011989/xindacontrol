package merchant

import (
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"
)

// ========== 商户 OSS 配置管理 ==========

// MerchantOssConfigReq 商户 OSS 配置请求（引用云账号，只存储 bucket 信息）
type MerchantOssConfigReq struct {
	Id             int    `json:"id"`
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id" binding:"required"` // 引用 cloud_accounts 表
	Name           string `json:"name" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	Region         string `json:"region"`
	Endpoint       string `json:"endpoint"`       // 可选，留空自动生成
	CustomDomain   string `json:"custom_domain"`  // CDN 域名
	IsDefault      int    `json:"is_default"`
	Status         int    `json:"status"`
}

// MerchantOssConfigResp 商户 OSS 配置响应
type MerchantOssConfigResp struct {
	Id               int    `json:"id"`
	MerchantId       int    `json:"merchant_id"`
	CloudAccountId   int64  `json:"cloud_account_id"`
	CloudAccountName string `json:"cloud_account_name"` // 云账号名称
	CloudType        string `json:"cloud_type"`         // 云类型 (从云账号获取)
	Name             string `json:"name"`
	Bucket           string `json:"bucket"`
	Region           string `json:"region"`
	Endpoint         string `json:"endpoint"`
	CustomDomain     string `json:"custom_domain"`
	IsDefault        int    `json:"is_default"`
	Status           int    `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// ListMerchantOssConfigs 获取商户的 OSS 配置列表
func ListMerchantOssConfigs(merchantId int) ([]MerchantOssConfigResp, error) {
	var configs []entity.MerchantOssConfigs
	err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).OrderBy("is_default DESC, id ASC").Find(&configs)
	if err != nil {
		return nil, err
	}

	// 获取关联的云账号信息
	accountIds := make([]int64, 0, len(configs))
	for _, c := range configs {
		accountIds = append(accountIds, c.CloudAccountId)
	}

	accountMap := make(map[int64]entity.CloudAccounts)
	if len(accountIds) > 0 {
		var accounts []entity.CloudAccounts
		err = dbs.DBAdmin.In("id", accountIds).Find(&accounts)
		if err != nil {
			return nil, err
		}
		for _, a := range accounts {
			accountMap[a.Id] = a
		}
	}

	result := make([]MerchantOssConfigResp, len(configs))
	for i, c := range configs {
		account := accountMap[c.CloudAccountId]
		endpoint := c.Endpoint
		if endpoint == "" {
			endpoint = generateOssEndpoint(account.CloudType, account.SiteType, c.Region)
		}
		result[i] = MerchantOssConfigResp{
			Id:               c.Id,
			MerchantId:       c.MerchantId,
			CloudAccountId:   c.CloudAccountId,
			CloudAccountName: account.Name,
			CloudType:        account.CloudType,
			Name:             c.Name,
			Bucket:           c.Bucket,
			Region:           c.Region,
			Endpoint:         endpoint,
			CustomDomain:     c.CustomDomain,
			IsDefault:        c.IsDefault,
			Status:           c.Status,
			CreatedAt:        c.CreatedAt.Format(time.DateTime),
			UpdatedAt:        c.UpdatedAt.Format(time.DateTime),
		}
	}
	return result, nil
}

// generateOssEndpoint 根据云类型和区域自动生成 OSS Endpoint
func generateOssEndpoint(cloudType, siteType, region string) string {
	if region == "" {
		return ""
	}
	switch cloudType {
	case "aliyun":
		if siteType == "intl" {
			return fmt.Sprintf("oss-%s.aliyuncs.com", region)
		}
		return fmt.Sprintf("oss-%s.aliyuncs.com", region)
	case "tencent":
		return fmt.Sprintf("cos.%s.myqcloud.com", region)
	case "aws":
		return fmt.Sprintf("s3.%s.amazonaws.com", region)
	default:
		return ""
	}
}

// CreateMerchantOssConfig 创建商户 OSS 配置
func CreateMerchantOssConfig(req MerchantOssConfigReq) (int, error) {
	// 检查云账号是否存在
	var account entity.CloudAccounts
	has, err := dbs.DBAdmin.ID(req.CloudAccountId).Get(&account)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, fmt.Errorf("云账号不存在")
	}

	// 如果设置为默认，先清除其他默认
	if req.IsDefault == 1 {
		_, err := dbs.DBAdmin.Exec("UPDATE merchant_oss_configs SET is_default = 0 WHERE merchant_id = ?", req.MerchantId)
		if err != nil {
			return 0, err
		}
	}

	config := &entity.MerchantOssConfigs{
		MerchantId:     req.MerchantId,
		CloudAccountId: req.CloudAccountId,
		Name:           req.Name,
		Bucket:         req.Bucket,
		Region:         req.Region,
		Endpoint:       req.Endpoint,
		CustomDomain:   req.CustomDomain,
		IsDefault:      req.IsDefault,
		Status:         1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = dbs.DBAdmin.Insert(config)
	if err != nil {
		return 0, err
	}
	return config.Id, nil
}

// UpdateMerchantOssConfig 更新商户 OSS 配置
func UpdateMerchantOssConfig(id int, req MerchantOssConfigReq) error {
	// 检查是否存在
	var config entity.MerchantOssConfigs
	has, err := dbs.DBAdmin.ID(id).Get(&config)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("OSS 配置不存在")
	}

	// 如果设置为默认，先清除其他默认
	if req.IsDefault == 1 {
		_, err := dbs.DBAdmin.Exec("UPDATE merchant_oss_configs SET is_default = 0 WHERE merchant_id = ? AND id != ?", config.MerchantId, id)
		if err != nil {
			return err
		}
	}

	updates := map[string]interface{}{
		"name":          req.Name,
		"bucket":        req.Bucket,
		"region":        req.Region,
		"endpoint":      req.Endpoint,
		"custom_domain": req.CustomDomain,
		"is_default":    req.IsDefault,
		"updated_at":    time.Now(),
	}

	// 如果提供了新的云账号ID，更新它
	if req.CloudAccountId > 0 {
		// 验证云账号存在
		has, err := dbs.DBAdmin.ID(req.CloudAccountId).Exist(&entity.CloudAccounts{})
		if err != nil {
			return err
		}
		if !has {
			return fmt.Errorf("云账号不存在")
		}
		updates["cloud_account_id"] = req.CloudAccountId
	}

	if req.Status != 0 {
		updates["status"] = req.Status
	}

	_, err = dbs.DBAdmin.Table("merchant_oss_configs").Where("id = ?", id).Update(updates)
	return err
}

// DeleteMerchantOssConfig 删除商户 OSS 配置
func DeleteMerchantOssConfig(id int) error {
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.MerchantOssConfigs{})
	return err
}

// GetMerchantDefaultOss 获取商户默认 OSS 配置（包含云账号凭证）
func GetMerchantDefaultOss(merchantId int) (*entity.MerchantOssConfigs, *entity.CloudAccounts, error) {
	var config entity.MerchantOssConfigs
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND is_default = 1 AND status = 1", merchantId).Get(&config)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		// 如果没有默认的，取第一个启用的
		has, err = dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).OrderBy("id ASC").Get(&config)
		if err != nil {
			return nil, nil, err
		}
		if !has {
			return nil, nil, nil
		}
	}

	// 获取关联的云账号
	var account entity.CloudAccounts
	has, err = dbs.DBAdmin.ID(config.CloudAccountId).Get(&account)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, nil, fmt.Errorf("关联的云账号不存在")
	}

	return &config, &account, nil
}

// ========== 商户 GOST 服务器关联管理 ==========

// MerchantGostServerReq 商户 GOST 服务器关联请求
type MerchantGostServerReq struct {
	Id         int    `json:"id"`
	MerchantId int    `json:"merchant_id" binding:"required"`
	ServerId   int    `json:"server_id" binding:"required"`
	CloudType  string `json:"cloud_type"`
	Region     string `json:"region"`
	ListenPort int    `json:"listen_port"`
	IsPrimary  int    `json:"is_primary"`
	Priority   int    `json:"priority"`
	Status     int    `json:"status"`
	Remark     string `json:"remark"`
}

// MerchantGostServerResp 商户 GOST 服务器关联响应
type MerchantGostServerResp struct {
	Id           int    `json:"id"`
	MerchantId   int    `json:"merchant_id"`
	ServerId     int    `json:"server_id"`
	ServerName   string `json:"server_name"`
	ServerHost   string `json:"server_host"`
	CloudType    string `json:"cloud_type"`
	Region       string `json:"region"`
	ListenPort   int    `json:"listen_port"`
	IsPrimary    int    `json:"is_primary"`
	Priority     int    `json:"priority"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
	ServerStatus int    `json:"server_status"` // 服务器状态
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ListMerchantGostServers 获取商户的 GOST 服务器列表
func ListMerchantGostServers(merchantId int) ([]MerchantGostServerResp, error) {
	var relations []entity.MerchantGostServers
	err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).OrderBy("is_primary DESC, priority ASC, id ASC").Find(&relations)
	if err != nil {
		return nil, err
	}

	// 获取关联的服务器信息
	serverIds := make([]int, len(relations))
	for i, r := range relations {
		serverIds[i] = r.ServerId
	}

	serverMap := make(map[int]entity.Servers)
	if len(serverIds) > 0 {
		var servers []entity.Servers
		err = dbs.DBAdmin.In("id", serverIds).Find(&servers)
		if err != nil {
			return nil, err
		}
		for _, s := range servers {
			serverMap[s.Id] = s
		}
	}

	result := make([]MerchantGostServerResp, len(relations))
	for i, r := range relations {
		server := serverMap[r.ServerId]
		result[i] = MerchantGostServerResp{
			Id:           r.Id,
			MerchantId:   r.MerchantId,
			ServerId:     r.ServerId,
			ServerName:   server.Name,
			ServerHost:   server.Host,
			CloudType:    r.CloudType,
			Region:       r.Region,
			ListenPort:   r.ListenPort,
			IsPrimary:    r.IsPrimary,
			Priority:     r.Priority,
			Status:       r.Status,
			Remark:       r.Remark,
			ServerStatus: server.Status,
			CreatedAt:    r.CreatedAt.Format(time.DateTime),
			UpdatedAt:    r.UpdatedAt.Format(time.DateTime),
		}
	}
	return result, nil
}

// CreateMerchantGostServer 创建商户 GOST 服务器关联
func CreateMerchantGostServer(req MerchantGostServerReq) (int, error) {
	// 检查服务器是否存在
	var server entity.Servers
	has, err := dbs.DBAdmin.ID(req.ServerId).Get(&server)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, fmt.Errorf("服务器不存在")
	}

	// 检查是否已关联
	var existing entity.MerchantGostServers
	has, err = dbs.DBAdmin.Where("merchant_id = ? AND server_id = ?", req.MerchantId, req.ServerId).Get(&existing)
	if err != nil {
		return 0, err
	}
	if has {
		return 0, fmt.Errorf("该服务器已关联此商户")
	}

	// 如果设置为主服务器，先清除其他主服务器
	if req.IsPrimary == 1 {
		_, err := dbs.DBAdmin.Exec("UPDATE merchant_gost_servers SET is_primary = 0 WHERE merchant_id = ?", req.MerchantId)
		if err != nil {
			return 0, err
		}
	}

	relation := &entity.MerchantGostServers{
		MerchantId: req.MerchantId,
		ServerId:   req.ServerId,
		CloudType:  req.CloudType,
		Region:     req.Region,
		ListenPort: req.ListenPort,
		IsPrimary:  req.IsPrimary,
		Priority:   req.Priority,
		Status:     1,
		Remark:     req.Remark,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err = dbs.DBAdmin.Insert(relation)
	if err != nil {
		return 0, err
	}
	return relation.Id, nil
}

// UpdateMerchantGostServer 更新商户 GOST 服务器关联
func UpdateMerchantGostServer(id int, req MerchantGostServerReq) error {
	var relation entity.MerchantGostServers
	has, err := dbs.DBAdmin.ID(id).Get(&relation)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("关联记录不存在")
	}

	// 如果设置为主服务器，先清除其他主服务器
	if req.IsPrimary == 1 {
		_, err := dbs.DBAdmin.Exec("UPDATE merchant_gost_servers SET is_primary = 0 WHERE merchant_id = ? AND id != ?", relation.MerchantId, id)
		if err != nil {
			return err
		}
	}

	updates := map[string]interface{}{
		"cloud_type":  req.CloudType,
		"region":      req.Region,
		"listen_port": req.ListenPort,
		"is_primary":  req.IsPrimary,
		"priority":    req.Priority,
		"remark":      req.Remark,
		"updated_at":  time.Now(),
	}

	if req.Status != 0 {
		updates["status"] = req.Status
	}

	_, err = dbs.DBAdmin.Table("merchant_gost_servers").Where("id = ?", id).Update(updates)
	return err
}

// DeleteMerchantGostServer 删除商户 GOST 服务器关联
func DeleteMerchantGostServer(id int) error {
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.MerchantGostServers{})
	return err
}

// GetMerchantPrimaryGostServer 获取商户主 GOST 服务器
func GetMerchantPrimaryGostServer(merchantId int) (*entity.MerchantGostServers, *entity.Servers, error) {
	var relation entity.MerchantGostServers
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND is_primary = 1 AND status = 1", merchantId).Get(&relation)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		// 如果没有主服务器，取优先级最高的
		has, err = dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).OrderBy("priority ASC, id ASC").Get(&relation)
		if err != nil {
			return nil, nil, err
		}
		if !has {
			return nil, nil, nil
		}
	}

	var server entity.Servers
	has, err = dbs.DBAdmin.ID(relation.ServerId).Get(&server)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, nil, fmt.Errorf("服务器不存在")
	}

	return &relation, &server, nil
}

// GetMerchantAllGostServers 获取商户所有启用的 GOST 服务器
func GetMerchantAllGostServers(merchantId int) ([]entity.MerchantGostServers, []entity.Servers, error) {
	var relations []entity.MerchantGostServers
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).OrderBy("is_primary DESC, priority ASC").Find(&relations)
	if err != nil {
		return nil, nil, err
	}

	if len(relations) == 0 {
		return nil, nil, nil
	}

	serverIds := make([]int, len(relations))
	for i, r := range relations {
		serverIds[i] = r.ServerId
	}

	var servers []entity.Servers
	err = dbs.DBAdmin.In("id", serverIds).Where("status = 1").Find(&servers)
	if err != nil {
		return nil, nil, err
	}

	return relations, servers, nil
}
