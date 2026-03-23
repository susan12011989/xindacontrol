package model

// ========== 服务器管理 ==========

// 查询服务器请求
type QueryServersReq struct {
	Pagination
	Name       string `json:"name" form:"name"`
	Host       string `json:"host" form:"host"`
	Status     *int   `json:"status" form:"status"`
	ServerType *int   `json:"server_type" form:"server_type"` // 1-商户服务器 2-系统服务器
	MerchantId *int   `json:"merchant_id" form:"merchant_id"` // 按商户ID筛选
	GroupId    *int   `json:"group_id" form:"group_id"`       // 按分组ID筛选
}

// 创建服务器请求
type CreateServerReq struct {
	Name        string `json:"name" binding:"required"`
	Host        string `json:"host" binding:"required"`
	AuxiliaryIP string `json:"auxiliary_ip"` // 辅助IP，仅系统服务器使用
	Port        int    `json:"port" binding:"required"`
	Username    string `json:"username" binding:"required"`
	AuthType    int    `json:"auth_type" binding:"required,oneof=1 2"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	ServerType  int    `json:"server_type"`  // 1-商户服务器 2-系统服务器
	ForwardType int    `json:"forward_type"` // 转发类型：1-加密(relay+tls) 2-直连(tcp)，仅系统服务器有效
	MerchantId  int    `json:"merchant_id"`  // 关联的商户ID
	GroupId     int    `json:"group_id"`     // 分组ID
	Description string `json:"description"`
}

// 更新服务器请求
type UpdateServerReq struct {
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	AuxiliaryIP *string `json:"auxiliary_ip"` // 辅助IP，仅系统服务器使用
	Port        *int    `json:"port"`
	Username    string  `json:"username"`
	AuthType    *int    `json:"auth_type"`
	Password    string  `json:"password"`
	PrivateKey  string  `json:"private_key"`
	ServerType  *int    `json:"server_type"`  // 1-商户服务器 2-系统服务器
	ForwardType *int    `json:"forward_type"` // 转发类型：1-加密(relay+tls) 2-直连(tcp)，仅系统服务器有效
	MerchantId  *int    `json:"merchant_id"`  // 关联的商户ID（指针区分未传和清零）
	GroupId     *int    `json:"group_id"`     // 分组ID（指针区分未传和清零）
	Status      *int    `json:"status"`
	Description string  `json:"description"`
}

// 服务器响应
type ServerResp struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	AuxiliaryIP  string `json:"auxiliary_ip"` // 辅助IP，仅系统服务器使用
	Port         int    `json:"port"`
	Username     string `json:"username"`
	AuthType     int    `json:"auth_type"`
	ServerType   int    `json:"server_type"`  // 1-商户服务器 2-系统服务器
	ForwardType  int    `json:"forward_type"` // 转发类型：1-加密(relay+tls) 2-直连(tcp)
	Status        int    `json:"status"`
	TlsEnabled    int    `json:"tls_enabled"`     // 客户端TLS：0-未启用 1-已启用
	TlsDeployedAt string `json:"tls_deployed_at"` // TLS证书部署时间
	Description   string `json:"description"`
	MerchantId    int    `json:"merchant_id"`   // 关联的商户ID
	MerchantName string `json:"merchant_name"` // 关联的商户名称
	MerchantNo   string `json:"merchant_no"`   // 商户号
	GroupId      int    `json:"group_id"`      // 分组ID
	GroupName    string `json:"group_name"`    // 分组名称
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// 服务器列表响应
type QueryServersResponse struct {
	List  []ServerResp `json:"list"`
	Total int          `json:"total"`
}

// 测试连接请求
type TestConnectionReq struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	Username   string `json:"username" binding:"required"`
	AuthType   int    `json:"auth_type" binding:"required"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
}

// ========== 服务操作（systemctl） ==========

// 支持的服务列表：server, wukongim, gost
var SupportedServices = []string{"server", "wukongim", "gost"}

// 服务上传路径映射（Docker 部署：宿主机上的二进制所在目录）
var ServiceUploadPaths = map[string]string{
	"server":   "/opt/tsdd/",
	"wukongim": "/root/wukongim/",
}

// 服务可执行文件名映射（Docker 部署：宿主机上挂载到容器内的二进制文件名）
var ServiceBinaryNames = map[string]string{
	"server":   "TangSengDaoDaoServer", // 挂载到容器 /home/app
	"wukongim": "WuKongIM",
}

// 服务 Docker 容器名映射
var ServiceDockerNames = map[string]string{
	"server":   "tsdd-server",
	"wukongim": "tsdd-wukongim",
	"gost":     "gost",
}

// 服务操作请求
type ServiceActionReq struct {
	ServerId    int    `json:"server_id" binding:"required"`
	ServiceName string `json:"service_name" binding:"required"` // server, wukongim, gost
	Action      string `json:"action" binding:"required,oneof=start stop restart"`
}

// 服务操作响应
type ServiceActionResp struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error_msg"`
}

// 服务状态请求
type ServiceStatusReq struct {
	ServerId    int    `json:"server_id" form:"server_id" binding:"required"`
	ServiceName string `json:"service_name" form:"service_name"` // 可选，为空则查询所有服务
}

// 服务状态响应
type ServiceStatusResp struct {
	ServiceName string `json:"service_name"`
	Status      string `json:"status"` // running/stopped/unknown
	Pid         int    `json:"pid"`
	Uptime      string `json:"uptime"`
	CPU         string `json:"cpu"`
	Memory      string `json:"memory"`
}

// 服务状态列表响应
type ServiceStatusListResp struct {
	Services []ServiceStatusResp `json:"services"`
}

// 服务日志请求
type ServiceLogsReq struct {
	ServerId    int    `json:"server_id" form:"server_id" binding:"required"`
	ServiceName string `json:"service_name" form:"service_name" binding:"required"`
	Lines       int    `json:"lines" form:"lines"` // 显示行数，默认100
}

// 服务日志响应
type ServiceLogsResp struct {
	Logs        string `json:"logs"`
	TotalLines  int    `json:"total_lines"`
	ServiceName string `json:"service_name"`
}

// ========== 服务器资源 ==========

// 服务器资源响应
type ServerStatsResp struct {
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	MemoryTotal string `json:"memory_total"`
	DiskUsage   string `json:"disk_usage"`
	DiskTotal   string `json:"disk_total"`
	LoadAvg     string `json:"load_avg"`
}

// 获取服务器资源请求
type GetServerStatsReq struct {
	ServerId int `json:"server_id" form:"server_id" binding:"required"`
}

// 批量获取服务器资源请求
type GetServerStatsBatchReq struct {
	ServerIds []int `json:"server_ids" binding:"required"`
}

// 批量服务器基础资源
type ServerBasicStat struct {
	ServerId    int    `json:"server_id"`
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	MemoryTotal string `json:"memory_total"`
	Error       string `json:"error,omitempty"`
}

// 批量获取服务器资源响应
type GetServerStatsBatchResp struct {
	Stats []ServerBasicStat `json:"stats"`
}

// ========== 隧道连接监控 ==========

// 隧道统计请求
type GetTrafficStatsReq struct {
	ServerId int `json:"server_id" form:"server_id" binding:"required"`
}

// 单条隧道统计
type TunnelStatItem struct {
	Name         string `json:"name"`
	Port         int    `json:"port"`
	Target       string `json:"target"`
	CurrentConns int    `json:"current_conns"`
	TotalConns   int64  `json:"total_conns"`
	InputBytes   int64  `json:"input_bytes"`
	OutputBytes  int64  `json:"output_bytes"`
	TotalErrs    int64  `json:"total_errs"`
	State        string `json:"state"`
}

// 隧道统计响应
type TunnelStatsResp struct {
	Tunnels           []TunnelStatItem `json:"tunnels"`
	TotalTunnels      int              `json:"total_tunnels"`
	TotalCurrentConns int              `json:"total_current_conns"`
	IdleTunnels       int              `json:"idle_tunnels"` // 当前无连接的隧道数
	AlertLevel        string           `json:"alert_level"`
	AlertMsg          string           `json:"alert_msg"`
}

// 批量请求
type GetTrafficStatsBatchReq struct {
	ServerIds []int `json:"server_ids" binding:"required"`
}

// 批量概要
type ServerTunnelSummary struct {
	ServerId          int    `json:"server_id"`
	TotalTunnels      int    `json:"total_tunnels"`
	TotalCurrentConns int    `json:"total_current_conns"`
	IdleTunnels       int    `json:"idle_tunnels"`
	AlertLevel        string `json:"alert_level"`
	AlertMsg          string `json:"alert_msg"`
	Error             string `json:"error,omitempty"`
}

// 批量响应
type TunnelStatsBatchResp struct {
	Stats []ServerTunnelSummary `json:"stats"`
}

// 历史记录查询请求
type TunnelStatsHistoryReq struct {
	ServerId   int    `json:"server_id" form:"server_id"`
	MerchantId int    `json:"merchant_id" form:"merchant_id"`
	TunnelName string `json:"tunnel_name" form:"tunnel_name"`
	StartTime  string `json:"start_time" form:"start_time"`
	EndTime    string `json:"end_time" form:"end_time"`
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"page_size" form:"page_size"`
}

// 历史记录项
type TunnelStatsHistoryItem struct {
	ServerId     int    `json:"server_id"`
	ServerHost   string `json:"server_host"`
	MerchantId   int    `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	TunnelName   string `json:"tunnel_name"`
	Port         int    `json:"port"`
	Target       string `json:"target"`
	CurrentConns int    `json:"current_conns"`
	TotalConns   int64  `json:"total_conns"`
	InputBytes   int64  `json:"input_bytes"`
	OutputBytes  int64  `json:"output_bytes"`
	TotalErrs    int64  `json:"total_errs"`
	State        string `json:"state"`
	CollectedAt  string `json:"collected_at"`
}

// 历史记录响应
type TunnelStatsHistoryResp struct {
	Total   int                      `json:"total"`
	Records []TunnelStatsHistoryItem `json:"records"`
}

// 商户隧道汇总（主视图）
type MerchantTunnelSummary struct {
	MerchantId   int    `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	ServerIP     string `json:"server_ip"`
	ServerId     int    `json:"server_id"`
	ForwardState string `json:"forward_state"` // online/offline/partial
	CurrentConns int    `json:"current_conns"`
	TotalConns   int64  `json:"total_conns"`
	TunnelCount  int    `json:"tunnel_count"`
	IdleCount    int    `json:"idle_count"`    // 无连接的隧道数
	ErrorCount   int64  `json:"error_count"`
	AlertLevel   string `json:"alert_level"`
}

// 商户隧道汇总响应
type MerchantTunnelOverviewResp struct {
	Merchants []MerchantTunnelSummary `json:"merchants"`
	Total     int                     `json:"total"`
}

// 隧道统计聚合查询请求
type TunnelStatsAggregateReq struct {
	MerchantId int    `json:"merchant_id" form:"merchant_id"`           // 按商户筛选
	ServerId   int    `json:"server_id" form:"server_id"`               // 按服务器筛选
	StartTime  string `json:"start_time" form:"start_time"`             // 开始时间 2026-03-01
	EndTime    string `json:"end_time" form:"end_time"`                 // 结束时间 2026-03-23
	GroupBy    string `json:"group_by" form:"group_by"`                 // 聚合粒度: hour/day（默认 day）
}

// 聚合统计项
type TunnelStatsAggregateItem struct {
	TimeSlot     string `json:"time_slot"`     // 时间段 2026-03-20 或 2026-03-20 14:00
	MerchantName string `json:"merchant_name"`
	ServerIP     string `json:"server_ip"`
	MaxConns     int    `json:"max_conns"`     // 该时段最大连接数
	AvgConns     int    `json:"avg_conns"`     // 该时段平均连接数
	TotalConns   int64  `json:"total_conns"`   // 该时段累计连接数
	TotalInput   int64  `json:"total_input"`   // 该时段入站字节数
	TotalOutput  int64  `json:"total_output"`  // 该时段出站字节数
	TotalErrs    int64  `json:"total_errs"`    // 该时段错误数
	SampleCount  int    `json:"sample_count"`  // 采样次数
}

// 聚合统计响应
type TunnelStatsAggregateResp struct {
	Items []TunnelStatsAggregateItem `json:"items"`
	Total int                        `json:"total"`
}

// 应急封禁请求
type BlockIPReq struct {
	ServerId int    `json:"server_id" binding:"required"`
	IP       string `json:"ip" binding:"required"`
	Duration string `json:"duration"` // "1h", "24h", "permanent"
}

// 应急限流请求
type EmergencyRateLimitReq struct {
	ServerId     int `json:"server_id" binding:"required"`
	MaxConnPerIP int `json:"max_conn_per_ip"` // 0=取消限制
	MaxSynRate   int `json:"max_syn_rate"`    // 0=取消限制
}

// ========== 配置文件 ==========

// 获取配置文件请求
type GetConfigFileReq struct {
	ServerId    int    `json:"server_id" form:"server_id" binding:"required"`
	ServiceName string `json:"service_name" form:"service_name" binding:"required"`
}

// 配置文件响应
type ConfigFileResp struct {
	ServiceName string `json:"service_name"`
	ConfigPath  string `json:"config_path"`
	Content     string `json:"content"`
}

// 更新配置文件请求
type UpdateConfigFileReq struct {
	ServerId    int    `json:"server_id" binding:"required"`
	ServiceName string `json:"service_name" binding:"required"`
	Content     string `json:"content" binding:"required"`
}

// ========== GOST API 代理 ==========

// 创建 GOST 服务请求
type CreateGostServiceReq struct {
	ServerId    int    `json:"server_id" binding:"required"`
	ListenPort  int    `json:"listen_port" binding:"required"`
	ForwardHost string `json:"forward_host" binding:"required"`
	ForwardPort int    `json:"forward_port" binding:"required"`
}

// 更新 GOST 服务请求
type UpdateGostServiceReq struct {
	Config interface{} `json:"config" binding:"required"`
}

// ========== 批量分发 ==========

// 批量分发请求（从本地服务器分发到目标服务器）
type DistributeFileReq struct {
	ServiceName     string `json:"service_name" binding:"required"`      // 服务名：server 或 wukongim
	TargetServerIds []int  `json:"target_server_ids" binding:"required"` // 目标服务器ID列表（商户服务器）
	RestartAfter    bool   `json:"restart_after"`                        // 分发后是否重启服务
}

// 本地上传路径（控制后台所在服务器）
var LocalUploadDir = "/tmp/deploy_uploads"

// 单个服务器分发结果
type DistributeResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

// ========== Docker 容器状态 ==========

// Docker 容器状态请求
type DockerContainersReq struct {
	ServerId int `json:"server_id" form:"server_id" binding:"required"`
}

// Docker 容器状态
type DockerContainerStatus struct {
	ContainerId string `json:"container_id"` // 容器ID（短）
	Name        string `json:"name"`         // 容器名称
	Image       string `json:"image"`        // 镜像名
	Status      string `json:"status"`       // 状态（Up/Exited等）
	Ports       string `json:"ports"`        // 端口映射
	Created     string `json:"created"`      // 创建时间
	RunningFor  string `json:"running_for"`  // 运行时长
	CPUPercent  string `json:"cpu_percent"`  // CPU 使用率
	MemUsage    string `json:"mem_usage"`    // 内存使用
	MemPercent  string `json:"mem_percent"`  // 内存使用率
}

// Docker 容器状态响应
type DockerContainersResp struct {
	Containers []DockerContainerStatus `json:"containers"`
}

// 批量分发响应
type DistributeFileResp struct {
	TotalCount   int                `json:"total_count"`
	SuccessCount int                `json:"success_count"`
	FailCount    int                `json:"fail_count"`
	Results      []DistributeResult `json:"results"`
}

// ========== GOST 服务器一键部署 ==========

// 部署 GOST 服务器请求
type DeployGostServerReq struct {
	CloudAccountId int64  `json:"cloud_account_id" binding:"required"` // 云账号ID
	RegionId       string `json:"region_id" binding:"required"`        // 地区ID
	InstanceType   string `json:"instance_type"`                       // 实例类型，为空使用默认
	ImageId        string `json:"image_id"`                            // 镜像ID，为空使用默认Ubuntu
	ServerName     string `json:"server_name"`                         // 服务器名称
	GroupId        int    `json:"group_id"`                            // 服务器分组ID
	Password       string `json:"password"`                            // SSH 密码（可选，不填则自动生成密钥）
	Bandwidth      string `json:"bandwidth"`                           // EIP 带宽，默认 5Mbps
}

// 在已有服务器上安装 GOST 请求
type InstallGostReq struct {
	ServerId   int    `json:"server_id"`   // 服务器ID（二选一）
	Host       string `json:"host"`        // 服务器IP（二选一）
	Port       int    `json:"port"`        // SSH端口，默认22
	Username   string `json:"username"`    // SSH用户名，默认root
	Password   string `json:"password"`    // SSH密码
	PrivateKey string `json:"private_key"` // SSH私钥（二选一）
}

// ========== GOST 转发配置（一键部署） ==========

// 配置 GOST 转发请求
type SetupGostForwardReq struct {
	ServerId int    `json:"server_id" binding:"required"` // GOST 服务器ID
	TargetIP string `json:"target_ip" binding:"required"` // 转发目标IP
	Ports    []int  `json:"ports"`                        // 转发端口列表（可选，为空使用默认）
	Mode     string `json:"mode"`                         // 连接模式：tls(加密，默认) 或 tcp(直连)
}

// 清除 GOST 转发请求
type ClearGostForwardReq struct {
	ServerId int   `json:"server_id" binding:"required"` // GOST 服务器ID
	Ports    []int `json:"ports"`                        // 要清除的端口列表（可选，为空清除所有）
}

// GOST 转发状态响应
type GostForwardStatusResp struct {
	ServerId    int                `json:"server_id"`
	ServerName  string             `json:"server_name"`
	ServerIP    string             `json:"server_ip"`
	Forwards    []GostForwardItem  `json:"forwards"`
	TotalCount  int                `json:"total_count"`
}

// 单个转发项
type GostForwardItem struct {
	Port     int    `json:"port"`      // 监听端口
	TargetIP string `json:"target_ip"` // 目标IP
	Status   string `json:"status"`    // 状态：active/inactive
}

// ========== Nginx 缓存管理 ==========

// 清除 Nginx 缓存请求
type ClearNginxCacheReq struct {
	ServerId int `json:"server_id" binding:"required"` // 系统服务器ID
}

// Nginx 缓存状态响应
type NginxCacheStatusResp struct {
	Installed bool   `json:"installed"` // Nginx 是否已安装
	Running   bool   `json:"running"`   // Nginx 是否运行中
	CacheSize string `json:"cache_size"` // 缓存目录大小，如 "156M"
}

// 安装 Nginx 请求
type InstallNginxReq struct {
	ServerId int `json:"server_id" binding:"required"` // 系统服务器ID
}

// ========== GOST 一键部署（安装+配置转发） ==========

// GOST 一键部署请求（安装+配置转发）
type SetupGostDeployReq struct {
	ServerId    int   `json:"server_id" binding:"required"`    // 系统服务器ID
	MerchantIds []int `json:"merchant_ids" binding:"required"` // 要配置转发的商户ID列表
	ForwardType int   `json:"forward_type"`                    // 转发类型: 1-加密(默认) 2-直连
}

// ========== GOST 配置持久化 ==========

// 持久化 GOST 配置请求
type PersistGostConfigReq struct {
	ServerId int `json:"server_id" binding:"required"`
}

// GOST 配置同步状态响应
type GostConfigSyncStatusResp struct {
	Synced              bool   `json:"synced"`                // 是否同步
	RunningServiceCount int    `json:"running_service_count"` // 运行中的服务数
	RunningChainCount   int    `json:"running_chain_count"`   // 运行中的链数
	FileServiceCount    int    `json:"file_service_count"`    // 文件中的服务数
	FileChainCount      int    `json:"file_chain_count"`      // 文件中的链数
	Message             string `json:"message"`               // 状态描述
}
