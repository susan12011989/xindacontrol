package model

// ========== 一键部署 TSDD 服务 ==========

// DeployTSDDReq 一键部署请求
type DeployTSDDReq struct {
	ServerId   int    `json:"server_id" binding:"required"`   // 目标服务器ID
	MerchantId int    `json:"merchant_id" binding:"required"` // 商户ID（用于获取配置）
	UseImages  bool   `json:"use_images"`                     // true=使用官方镜像, false=使用自定义二进制
	ForceReset bool   `json:"force_reset"`                    // 强制重置（删除现有容器和数据）
}

// DeployTSDDByIPReq 通过IP直接部署（新服务器，未注册到系统）
type DeployTSDDByIPReq struct {
	Host       string `json:"host" binding:"required"`        // 服务器IP
	Port       int    `json:"port"`                           // SSH端口，默认22
	Username   string `json:"username"`                       // SSH用户名，默认root
	Password   string `json:"password" binding:"required"`    // SSH密码
	MerchantId int    `json:"merchant_id" binding:"required"` // 商户ID
	ServerName string `json:"server_name"`                    // 服务器名称（用于注册）
	UseImages  bool   `json:"use_images"`                     // true=使用官方镜像
	ForceReset bool   `json:"force_reset"`                    // 强制重置
}

// DeployStep 部署步骤
type DeployStep struct {
	Name    string `json:"name"`    // 步骤名称
	Status  string `json:"status"`  // pending/running/success/failed
	Message string `json:"message"` // 详细信息
	Output  string `json:"output"`  // 命令输出
}

// DeployTSDDResp 部署响应
type DeployTSDDResp struct {
	Success  bool         `json:"success"`
	Message  string       `json:"message"`
	Steps    []DeployStep `json:"steps"`              // 各步骤执行结果
	ServerId int          `json:"server_id"`          // 服务器ID（新注册时返回）
	APIUrl   string       `json:"api_url,omitempty"`  // 部署后的API地址
	WebUrl   string       `json:"web_url,omitempty"`  // 部署后的Web地址
	AdminUrl string       `json:"admin_url,omitempty"` // 管理后台地址
}

// DeployConfig 部署配置
type DeployConfig struct {
	ExternalIP      string `json:"external_ip"`       // 外网IP
	MySQLPassword   string `json:"mysql_password"`    // MySQL密码
	MinioUser       string `json:"minio_user"`        // MinIO用户
	MinioPassword   string `json:"minio_password"`    // MinIO密码
	AdminPassword   string `json:"admin_password"`    // 管理后台密码
	SMSCode         string `json:"sms_code"`          // 测试短信验证码
	APIPort         int    `json:"api_port"`          // API端口，默认8090
	WSPort          int    `json:"ws_port"`           // WebSocket端口，默认5200
	WebPort         int    `json:"web_port"`          // Web端口，默认82
	ManagerPort     int    `json:"manager_port"`      // 管理后台端口，默认8084
}

// GetDeployStatusReq 获取部署状态请求
type GetDeployStatusReq struct {
	ServerId int `json:"server_id" form:"server_id" binding:"required"`
}

// GetDeployStatusResp 部署状态响应
type GetDeployStatusResp struct {
	Deployed    bool     `json:"deployed"`     // 是否已部署
	Services    []string `json:"services"`     // 已部署的服务列表
	Healthy     bool     `json:"healthy"`      // 服务是否健康
	LastDeploy  string   `json:"last_deploy"`  // 最后部署时间
	Version     string   `json:"version"`      // 部署版本
}

// 默认配置
var DefaultDeployConfig = DeployConfig{
	MySQLPassword: "TsddSecure2024!",
	MinioUser:     "admin",
	MinioPassword: "TsddMinio2024!",
	AdminPassword: "admin123",
	SMSCode:       "123456",
	APIPort:       8090,
	WSPort:        5200,
	WebPort:       82,
	ManagerPort:   8084,
}
