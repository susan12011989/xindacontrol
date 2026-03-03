package model

import "time"

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

// DeployNodeReq 集群节点部署请求（支持水平扩容）
type DeployNodeReq struct {
	ServerId   int    `json:"server_id"`                      // 目标服务器ID（为0时需创建EC2）
	MerchantId int    `json:"merchant_id" binding:"required"` // 商户ID
	NodeRole   string `json:"node_role" binding:"required"`   // 节点角色: allinone/db/minio/app
	ForceReset bool   `json:"force_reset"`                    // 强制重置

	// DB 连接（app 节点必填，指向 DB 节点内网 IP）
	DBHost string `json:"db_host"` // DB 节点内网 IP（MySQL+Redis）

	// MinIO 连接（app 节点必填，指向 MinIO 节点内网 IP；不填则与 DBHost 相同）
	MinioHost string `json:"minio_host"` // MinIO 节点内网 IP

	// WuKongIM 集群配置（app/allinone 节点）
	WKNodeId   int    `json:"wk_node_id"`   // WuKongIM 节点 ID（如 1001）
	WKSeedNode string `json:"wk_seed_node"` // 种子节点（加入已有集群时填写，如 "1001@172.31.0.1:11110"）

	// EC2 创建参数（填写 AmiId 时自动创建 EC2 + 注册服务器）
	AmiId          string `json:"ami_id"`           // AMI ID，填写后自动创建 EC2
	InstanceType   string `json:"instance_type"`    // EC2 实例类型，如 t3.large
	VolumeSizeGiB  int32  `json:"volume_size_gib"`  // 磁盘大小 GB
	CloudAccountId int64  `json:"cloud_account_id"` // AWS 云账号 ID
	RegionId       string `json:"region_id"`        // AWS 区域
	KeyName        string `json:"key_name"`         // SSH Key 名称
	SubnetId       string `json:"subnet_id"`        // 子网 ID
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
	ExternalIP    string `json:"external_ip"`    // 外网IP
	MySQLPassword string `json:"mysql_password"` // MySQL密码
	MinioUser     string `json:"minio_user"`     // MinIO用户
	MinioPassword string `json:"minio_password"` // MinIO密码
	AdminPassword string `json:"admin_password"` // 管理后台密码
	SMSCode       string `json:"sms_code"`       // 测试短信验证码
	APIPort       int    `json:"api_port"`       // API端口，默认8090
	WSPort        int    `json:"ws_port"`        // WebSocket端口，默认5200
	WebPort       int    `json:"web_port"`       // Web端口，默认82
	ManagerPort   int    `json:"manager_port"`   // 管理后台端口，默认8084

	// ===== 水平扩容相关 =====

	// NodeRole 节点角色: "allinone"(默认,全部服务), "db"(MySQL+Redis), "minio"(MinIO), "app"(WuKongIM+server)
	NodeRole string `json:"node_role"`

	// 远程 DB 连接（app 节点使用，指向 DB 节点内网 IP）
	DBHost    string `json:"db_host"`    // DB 节点内网 IP（空=localhost）
	RedisHost string `json:"redis_host"` // Redis 地址（空=localhost）
	MinioHost       string `json:"minio_host"`        // MinIO 内网地址（空=localhost）
	MinioPublicHost string `json:"minio_public_host"` // MinIO 公网地址（presigned URL 用，空=ExternalIP）

	// WuKongIM 集群配置
	WKNodeId   int    `json:"wk_node_id"`   // WuKongIM 集群节点 ID（如 1001, 1002）
	WKSeedNode string `json:"wk_seed_node"` // 种子节点（如 "1001@172.31.0.1:11110"，空=首个节点）

	// 节点自身内网 IP（WuKongIM 集群通信用，部署时自动检测）
	PrivateIP string `json:"private_ip"`

	// Control 面板回调（tsdd-server 上报状态用）
	ControlAPIUsername string `json:"control_api_username"`
	ControlAPIPassword string `json:"control_api_password"`

	// AMI 部署标记（跳过 Docker 安装和镜像拉取）
	FromAMI bool `json:"from_ami"`
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

// ClusterNodeInfo 集群节点信息（API 返回用）
type ClusterNodeInfo struct {
	Id         int        `json:"id"`
	MerchantId int        `json:"merchant_id"`
	ServerId   int        `json:"server_id"`
	ServerName string     `json:"server_name"`
	ServerHost string     `json:"server_host"`
	NodeRole   string     `json:"node_role"`
	PrivateIP  string     `json:"private_ip"`
	WKNodeId   int        `json:"wk_node_id"`
	DBHost     string     `json:"db_host"`
	MinioHost  string     `json:"minio_host"`
	Status     string     `json:"status"`
	DeployedAt *time.Time `json:"deployed_at"`
}

// GostPortForward 单个端口转发详情
type GostPortForward struct {
	Name       string `json:"name"`        // 协议名称: tcp/ws/http/minio
	ListenPort int    `json:"listen_port"` // GOST 监听端口（如 10000）
	TargetPort int    `json:"target_port"` // 转发目标端口（如 10010）
}

// GostSyncResult GOST 同步结果
type GostSyncResult struct {
	ServerId    int               `json:"server_id"`
	ServerName  string            `json:"server_name"`
	ServerHost  string            `json:"server_host"`
	TargetIP    string            `json:"target_ip"`
	ForwardType string            `json:"forward_type"` // encrypted/direct
	Ports       []GostPortForward `json:"ports"`        // 端口转发详情
	Success      bool              `json:"success"`
	Error        string            `json:"error,omitempty"`
	PersistError string            `json:"persist_error,omitempty"` // 持久化失败信息（转发成功但持久化失败时填充）
}

// SyncClusterGostReq 同步集群 GOST 转发请求
type SyncClusterGostReq struct {
	MerchantId int `json:"merchant_id" binding:"required"`
}

// ClusterWizardReq 集群一站式创建请求
type ClusterWizardReq struct {
	// 恢复模式：传入已有商户ID，跳过已完成的步骤（EC2已创建的角色、已部署的节点）
	MerchantId int `json:"merchant_id"`

	// 商户基本信息
	MerchantName string `json:"merchant_name"`
	AppName      string `json:"app_name"`
	Port         int    `json:"port"`
	ExpiredAt    string `json:"expired_at"`

	// AWS 配置（新建模式必填，恢复模式自动从已有数据获取）
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id"`
	KeyName        string `json:"key_name"`
	SubnetId       string `json:"subnet_id"`

	// 3 节点 EC2 配置（AMI 为空则使用默认 Ubuntu）
	DbAmiId            string `json:"db_ami_id"`             // DB 节点 AMI
	DbInstanceType     string `json:"db_instance_type"`      // 默认 r5.large
	DbVolumeSizeGiB    int32  `json:"db_volume_size_gib"`    // 默认 100
	MinioAmiId         string `json:"minio_ami_id"`          // MinIO 节点 AMI
	MinioInstanceType  string `json:"minio_instance_type"`   // 默认 t3.medium
	MinioVolumeSizeGiB int32  `json:"minio_volume_size_gib"` // 默认 200
	AppAmiId           string `json:"app_ami_id"`            // App 节点 AMI
	AppInstanceType    string `json:"app_instance_type"`     // 默认 t3.large
	AppVolumeSizeGiB   int32  `json:"app_volume_size_gib"`   // 默认 30
}

// ClusterWizardStep SSE 进度消息
type ClusterWizardStep struct {
	Step       int    `json:"step"`                    // 步骤编号 1-8
	Total      int    `json:"total"`                   // 总步骤数
	Title      string `json:"title"`                   // 步骤标题
	Status     string `json:"status"`                  // running/success/failed/skipped
	Message    string `json:"message"`                 // 详细信息
	MerchantId int    `json:"merchant_id,omitempty"`   // Step 1 成功时返回，前端用于重试
}

// 默认配置
var DefaultDeployConfig = DeployConfig{
	MySQLPassword: "TsddSecure2024!",
	MinioUser:     "admin",
	MinioPassword: "TsddMinio2024!",
	AdminPassword: "admin123",
	SMSCode:       "123456",
	APIPort:       10002,
	WSPort:        5200,
	WebPort:       82,
	ManagerPort:   8084,
}
