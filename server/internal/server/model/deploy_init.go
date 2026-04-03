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

// DeployNodeReq 集群节点部署请求（支持水平扩容）
type DeployNodeReq struct {
	ServerId   int    `json:"server_id" binding:"required"`   // 目标服务器ID
	MerchantId int    `json:"merchant_id" binding:"required"` // 商户ID
	NodeRole   string `json:"node_role" binding:"required"`   // 节点角色: allinone/db/app
	ForceReset bool   `json:"force_reset"`                    // 强制重置

	// DB/MinIO 连接（app 节点必填，指向内网 IP）
	DBHost    string `json:"db_host"`    // DB 节点内网 IP
	MinioHost string `json:"minio_host"` // MinIO 节点内网 IP（留空则同 DBHost）

	// WuKongIM 集群配置（app/allinone 节点）
	WKNodeId   int    `json:"wk_node_id"`   // WuKongIM 节点 ID（如 1001）
	WKSeedNode string `json:"wk_seed_node"` // 种子节点（加入已有集群时填写，如 "1001@172.31.0.1:11110"）

	// EC2 创建模式（server_id=0 时使用）
	AmiId          string `json:"ami_id"`
	InstanceType   string `json:"instance_type"`
	VolumeSizeGiB  int    `json:"volume_size_gib"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id"`
	KeyName        string `json:"key_name"`
	SubnetId       string `json:"subnet_id"`
}

// DeployConfig 部署配置
type DeployConfig struct {
	ExternalIP    string `json:"external_ip"`    // 外网IP
	MySQLPassword string `json:"mysql_password"` // MySQL密码
	MinioUser     string `json:"minio_user"`     // MinIO用户
	MinioPassword string `json:"minio_password"` // MinIO密码
	AdminPassword string `json:"admin_password"` // 管理后台密码
	SMSCode       string `json:"sms_code"`       // 测试短信验证码
	APIPort       int    `json:"api_port"`       // API端口，默认8090
	WSPort        int    `json:"ws_port"`        // WuKongIM WebSocket端口（Web端），默认5200
	WebPort       int    `json:"web_port"`       // Web端口，默认82
	ManagerPort   int    `json:"manager_port"`   // 管理后台端口，默认8084

	// ===== 集群部署相关 =====

	// NodeRole 节点角色: "allinone"(默认,全部服务), "db"(仅数据库), "app"(应用+WuKongIM)
	NodeRole string `json:"node_role"`

	// 远程 DB 连接（app 节点使用，指向 DB 节点内网 IP）
	DBHost    string `json:"db_host"`    // DB 节点内网 IP（空=localhost）
	RedisHost string `json:"redis_host"` // Redis 地址（空=localhost）
	MinioHost string `json:"minio_host"` // MinIO 地址（空=localhost）

	// WuKongIM 集群配置
	WKNodeId   int    `json:"wk_node_id"`   // WuKongIM 集群节点 ID（如 1001, 1002）
	WKSeedNode string `json:"wk_seed_node"` // 种子节点（如 "1001@172.31.0.1:11110"，空=首个节点）

	// 节点自身内网 IP（app 节点用于 WuKongIM 集群注册，不同于 DBHost）
	AppNodeIP string `json:"app_node_ip"`

	// Control 面板回调（tsdd-server 上报状态用）
	ControlAPIUsername string `json:"control_api_username"`
	ControlAPIPassword string `json:"control_api_password"`
}

// ClusterTopologyReq 集群拓扑查询
type ClusterTopologyReq struct {
	MerchantId int `json:"merchant_id" form:"merchant_id" binding:"required"`
}

// ClusterNodeInfo 集群节点信息
type ClusterNodeInfo struct {
	NodeId       int    `json:"node_id"`
	MerchantId   int    `json:"merchant_id"`
	Role         string `json:"role"`
	Host         string `json:"host"`
	PrivateIP    string `json:"private_ip"`
	ServerId     int    `json:"server_id"`
	ServerName   string `json:"server_name"`
	IsPrimary    int    `json:"is_primary"`
	Status       int    `json:"status"`
	WKNodeId     int    `json:"wk_node_id"`
	DBHost       string `json:"db_host"`
	MinioHost    string `json:"minio_host"`
	DeployStatus string `json:"deploy_status"`
	DeployError  string `json:"deploy_error"`
	LastDeployAt string `json:"last_deploy_at"`
}

// ClusterTopologyResp 集群拓扑响应
type ClusterTopologyResp struct {
	MerchantId   int               `json:"merchant_id"`
	MerchantName string            `json:"merchant_name"`
	DeployMode   string            `json:"deploy_mode"`
	Nodes        []ClusterNodeInfo `json:"nodes"`
}

// RetryDeployReq 重试部署请求
type RetryDeployReq struct {
	NodeId int `json:"node_id" binding:"required"`
}

// ClusterWizardReq 集群向导一键部署请求
type ClusterWizardReq struct {
	// 商户信息（新建时使用，恢复部署时可忽略）
	MerchantId   int    `json:"merchant_id"`   // >0 表示恢复已有商户的部署
	MerchantName string `json:"merchant_name"`
	AppName      string `json:"app_name"`
	Port         int    `json:"port"`
	ExpiredAt    string `json:"expired_at"`

	// AWS 配置
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id"`
	KeyName        string `json:"key_name"`
	SubnetId       string `json:"subnet_id"`

	// DB 节点（MySQL + Redis）
	DBAmiId         string `json:"db_ami_id"`
	DBInstanceType  string `json:"db_instance_type"`
	DBVolumeSizeGiB int    `json:"db_volume_size_gib"`

	// MinIO 节点
	MinIOAmiId         string `json:"minio_ami_id"`
	MinIOInstanceType  string `json:"minio_instance_type"`
	MinIOVolumeSizeGiB int    `json:"minio_volume_size_gib"`

	// App 节点（tsdd-server + WuKongIM + Web）
	AppAmiId         string `json:"app_ami_id"`
	AppInstanceType  string `json:"app_instance_type"`
	AppVolumeSizeGiB int    `json:"app_volume_size_gib"`
}

// ClusterWizardStepResp 集群向导流式步骤响应
type ClusterWizardStepResp struct {
	Step       int    `json:"step"`
	Total      int    `json:"total"`
	Title      string `json:"title"`
	Status     string `json:"status"`  // pending/running/success/failed/skipped
	Message    string `json:"message"`
	MerchantId int    `json:"merchant_id,omitempty"`
	Success    bool   `json:"success,omitempty"` // 最终完成标记
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
