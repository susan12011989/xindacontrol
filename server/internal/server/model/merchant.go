package model

import (
	"server/pkg/entity"
	"time"
)

// CreateOrEditMerchantReq 创建或编辑商户请求
type CreateOrEditMerchantReq struct {
	No                   string                       `json:"no"`        // 商户编号（创建时后端根据port计算，前端不再填写）
	Port                 int                          `json:"port"`      // 商户端口（创建必填）
	ServerIP             string                       `json:"server_ip"` // 服务器IP（创建必填）
	Name                 string                       `json:"name"`
	AppName              string                       `json:"app_name"`  // 应用名称（用于打包显示）
	LogoUrl              string                       `json:"logo_url"`  // Logo 地址
	IconUrl              string                       `json:"icon_url"`  // 应用图标地址
	Status               int                          `json:"status"`
	PackageConfiguration *entity.PackageConfiguration `json:"package_configuration"` // 套餐配置
	ExpiredAt            string                       `json:"expired_at"`
	// 创建商户时的AWS账号（仅创建时使用），会写入 cloud_accounts
	AwsAccessKeyId     string `json:"aws_access_key_id"`
	AwsAccessKeySecret string `json:"aws_access_key_secret"`
	// 选择现有系统AWS账号的ID（优先使用此选项）
	SelectedAwsAccountId int64 `json:"selected_aws_account_id"`
	// 是否将选中的系统账号移除（转为商户账号）
	RemoveFromSystem bool `json:"remove_from_system"`
}

// QueryMerchantReq 查询商户请求
type QueryMerchantReq struct {
	Pagination
	Name string `json:"name" form:"name"`
}

type Merchant struct {
	Id                   int                          `json:"id"`
	No                   string                       `json:"no"`   // 商户编号
	Port                 int                          `json:"port"` // 商户端口
	ServerIP             string                       `json:"server_ip"`
	Name                 string                       `json:"name"`
	AppName              string                       `json:"app_name"` // 应用名称
	LogoUrl              string                       `json:"logo_url"` // Logo 地址
	IconUrl              string                       `json:"icon_url"` // 应用图标地址
	Status               int                          `json:"status"`     // 1:正常,-1:禁用
	ExpiredAt            string                       `json:"expired_at"` // 服务过期时间
	CreatedAt            string                       `json:"created_at"`
	UpdatedAt            string                       `json:"updated_at"`
	PackageConfiguration *entity.PackageConfiguration `json:"package_configuration"` // 套餐配置
	Configs              *entity.Configs              `json:"configs,omitempty"`     // 全局配置
	AppConfigs           *entity.AppConfigs           `json:"app_configs,omitempty"` // 应用配置
	ExpiringSoon         int                          `json:"expiring_soon"`         // 2:已过期 1:即将过期 0:正常

	// 配置统计（列表增强）
	OssConfigCount  int `json:"oss_config_count"`  // OSS 配置数量
	GostServerCount int `json:"gost_server_count"` // GOST 服务器数量
}

func (m *Merchant) Init(e *entity.Merchants) {
	m.Id = e.Id
	m.No = e.No
	m.Port = e.Port
	m.ServerIP = e.ServerIP
	m.Name = e.Name
	m.AppName = e.AppName
	m.LogoUrl = e.LogoUrl
	m.IconUrl = e.IconUrl
	m.Status = e.Status
	m.PackageConfiguration = e.PackageConfiguration
	m.Configs = e.Configs
	m.AppConfigs = e.AppConfigs
	m.ExpiredAt = e.ExpiredAt.Format(time.DateTime)
	m.CreatedAt = e.CreatedAt.Format(time.DateTime)
	m.UpdatedAt = e.UpdatedAt.Format(time.DateTime)
	now := time.Now()
	if e.ExpiredAt.Before(now) {
		m.ExpiringSoon = 2
	} else if e.ExpiredAt.Before(now.AddDate(0, 0, 3)) {
		m.ExpiringSoon = 1
	}
}

type BalanceReq struct {
	MerchantId []int `form:"merchant_id[]" binding:"required"`
}
type BalanceData struct {
	MerchantId int    `json:"merchant_id"`
	Balance    string `json:"balance"`
}

// ========== 隧道连接检测 ==========

// TunnelCheckReq 隧道检测请求
// 可通过 merchant_id 获取商户 IP，或直接传 server_ip
type TunnelCheckReq struct {
	MerchantId int    `form:"merchant_id" json:"merchant_id"`
	ServerIP   string `form:"server_ip" json:"server_ip"`
}

// TunnelCheckItem 单台系统服务器的检测结果
type TunnelCheckItem struct {
	ServerName string `json:"server_name"`
	ServerIP   string `json:"server_ip"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

// 更换商户IP响应
type ChangeMerchantIPResp struct {
	OldIP         string `json:"old_ip"`
	NewIP         string `json:"new_ip"`
	Region        string `json:"region"`
	InstanceId    string `json:"instance_id"`
	OldAllocation string `json:"old_allocation_id,omitempty"`
	NewAllocation string `json:"new_allocation_id"`
}

// ChangeGostPortReq 更换商户 GOST 转发端口请求
type ChangeGostPortReq struct {
	GostPort int `json:"gost_port" binding:"required"` // 新的 GOST 监听/转发端口
}

// ChangeGostPortResp 更换商户 GOST 转发端口响应
type ChangeGostPortResp struct {
	MerchantId int `json:"merchant_id"`
	OldPort    int `json:"old_port"`
	NewPort    int `json:"new_port"`
}

// TunnelStats 隧道统计信息
type TunnelStats struct {
	TotalMerchants       int `json:"total_merchants"`        // 有效商户总数
	TotalGostServers     int `json:"total_gost_servers"`     // 系统服务器（GOST）总数
	TotalMerchantServers int `json:"total_merchant_servers"` // 商户服务器总数
}

// ========== TURN 服务器配置管理 ==========

// MerchantTurnConfigItem 商户 TURN 配置项
type MerchantTurnConfigItem struct {
	MerchantId     int    `json:"merchant_id"`
	MerchantNo     string `json:"merchant_no"`
	MerchantName   string `json:"merchant_name"`
	ServerIP       string `json:"server_ip"`
	Status         int    `json:"status"`
	TurnServer     string `json:"turn_server"`
	TurnUsername   string `json:"turn_username"`
	TurnCredential string `json:"turn_credential"`
	UpdatedAt      string `json:"updated_at"`
}

// BatchUpdateTurnServerReq 批量更新 TURN 请求
type BatchUpdateTurnServerReq struct {
	MerchantIds    []int  `json:"merchant_ids" binding:"required,min=1"`
	TurnServer     string `json:"turn_server" binding:"required"`
	TurnUsername   string `json:"turn_username"`
	TurnCredential string `json:"turn_credential"`
}

// BatchTurnUpdateResult 单商户更新结果
type BatchTurnUpdateResult struct {
	MerchantId   int    `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	ServerIP     string `json:"server_ip"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
}

// BatchUpdateTurnServerResp 批量更新响应
type BatchUpdateTurnServerResp struct {
	TotalCount   int                     `json:"total_count"`
	SuccessCount int                     `json:"success_count"`
	FailCount    int                     `json:"fail_count"`
	Results      []BatchTurnUpdateResult `json:"results"`
}
