package model

// ========== Docker 容器管理（实时查询） ==========

// 查询容器列表请求
type QueryDockerContainersReq struct {
	ServerId int    `json:"server_id" form:"server_id" binding:"required"`
	Status   string `json:"status" form:"status"` // all/running/exited
	Name     string `json:"name" form:"name"`     // 容器名称过滤（模糊匹配）
}

// 容器信息响应（从 docker ps 解析）
type DockerContainerResp struct {
	ContainerId string `json:"container_id"` // 容器ID（短ID）
	Name        string `json:"name"`         // 容器名称
	Image       string `json:"image"`        // 镜像
	Status      string `json:"status"`       // 状态文本：Up 2 hours / Exited (0) 5 days ago
	State       string `json:"state"`        // 状态：running/exited/paused/restarting
	Ports       string `json:"ports"`        // 端口映射
	CreatedAt   string `json:"created_at"`   // 创建时间
}

// 容器列表响应
type QueryDockerContainersResponse struct {
	List       []DockerContainerResp `json:"list"`
	Total      int                   `json:"total"`
	ServerInfo ServerInfo            `json:"server_info"` // 服务器信息
}

// 服务器信息
type ServerInfo struct {
	Id           int    `json:"id"`
	MerchantId   int    `json:"merchant_id"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	MerchantName string `json:"merchant_name"` // 关联查询商户名称
}

// 容器资源使用情况（从 docker stats 解析）
type DockerContainerStatsResp struct {
	ContainerId string `json:"container_id"`
	Name        string `json:"name"`
	CPUPerc     string `json:"cpu_perc"`  // CPU使用率：25.50%
	MemUsage    string `json:"mem_usage"` // 内存使用：1.5GiB / 8GiB
	MemPerc     string `json:"mem_perc"`  // 内存百分比：18.75%
	NetIO       string `json:"net_io"`    // 网络IO：1.2GB / 800MB
	BlockIO     string `json:"block_io"`  // 磁盘IO：100MB / 50MB
	Pids        string `json:"pids"`      // 进程数
}

// 容器详细信息请求
type GetDockerContainerDetailReq struct {
	ServerId    int    `json:"server_id" form:"server_id" binding:"required"`
	ContainerId string `json:"container_id" form:"container_id" binding:"required"`
}

// 容器日志请求
type GetDockerLogsReq struct {
	ServerId    int    `json:"server_id" form:"server_id" binding:"required"`
	ContainerId string `json:"container_id" form:"container_id" binding:"required"`
	Lines       int    `json:"lines" form:"lines"`           // 显示行数，默认100
	Since       string `json:"since" form:"since"`           // 开始时间：2024-01-01T00:00:00
	Until       string `json:"until" form:"until"`           // 结束时间：2024-01-02T00:00:00
	Follow      bool   `json:"follow" form:"follow"`         // 是否实时追踪（WebSocket）
	Timestamps  bool   `json:"timestamps" form:"timestamps"` // 是否显示时间戳
}

// 容器日志响应
type GetDockerLogsResponse struct {
	Logs          string `json:"logs"`           // 日志内容
	TotalLines    int    `json:"total_lines"`    // 总行数
	ContainerId   string `json:"container_id"`   // 容器ID
	ContainerName string `json:"container_name"` // 容器名称
}

// 容器操作请求
type DockerContainerOperationReq struct {
	ServerId    int    `json:"server_id" binding:"required"`
	ContainerId string `json:"container_id" binding:"required"`
	Action      string `json:"action" binding:"required,oneof=start stop restart remove"` // 操作类型
	Force       bool   `json:"force"`                                                     // 是否强制（用于删除）
}

// 批量操作请求
type DockerBatchOperationReq struct {
	ServerId     int      `json:"server_id" binding:"required"`
	ContainerIds []string `json:"container_ids" binding:"required,min=1"`
	Action       string   `json:"action" binding:"required,oneof=start stop restart remove"`
	Force        bool     `json:"force"`
}

// 操作结果响应
type DockerOperationResponse struct {
	Success bool                    `json:"success"`
	Message string                  `json:"message"`
	Results []DockerOperationResult `json:"results,omitempty"` // 批量操作时返回
}

// 单个操作结果
type DockerOperationResult struct {
	ContainerId string `json:"container_id"`
	Name        string `json:"name"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
}

// 查询 Docker 操作历史
type QueryDockerHistoryReq struct {
	Pagination
	ServerId    int    `json:"server_id" form:"server_id"`
	MerchantId  int    `json:"merchant_id" form:"merchant_id"`
	ContainerId string `json:"container_id" form:"container_id"`
	Action      string `json:"action" form:"action"`
}

// Docker 操作历史响应
type DockerHistoryResp struct {
	Id            int    `json:"id"`
	ServerName    string `json:"server_name"`
	MerchantName  string `json:"merchant_name"`
	ContainerId   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Action        string `json:"action"`
	Operator      string `json:"operator"`
	Status        int    `json:"status"`
	Output        string `json:"output"`
	ErrorMsg      string `json:"error_msg"`
	CreatedAt     string `json:"created_at"`
}

// Docker 操作历史列表响应
type QueryDockerHistoryResponse struct {
	List  []DockerHistoryResp `json:"list"`
	Total int                 `json:"total"`
}

// ========== 服务器健康检查 ==========

// 健康检查请求
type HealthCheckReq struct {
	ServerId int `json:"server_id" form:"server_id" binding:"required"`
}

// 单项健康检查结果
type HealthCheckItem struct {
	Name          string `json:"name"`                     // 检查项名称
	Status        string `json:"status"`                   // ok/error/warning
	Message       string `json:"message"`                  // 状态说明
	Latency       int64  `json:"latency"`                  // 响应时间(毫秒)
	Action        string `json:"action,omitempty"`         // 建议操作: restart/start/deploy/none
	ActionLabel   string `json:"action_label,omitempty"`   // 操作按钮文案
	ContainerName string `json:"container_name,omitempty"` // 关联的容器名称
}

// 健康检查响应
type HealthCheckResponse struct {
	ServerId   int               `json:"server_id"`
	ServerName string            `json:"server_name"`
	ServerHost string            `json:"server_host"`
	CheckTime  string            `json:"check_time"`
	Overall    string            `json:"overall"` // healthy/unhealthy/partial
	Services   []HealthCheckItem `json:"services"`
	APIs       []HealthCheckItem `json:"apis"`
}

// 批量健康检查请求
type BatchHealthCheckReq struct {
	ServerIds []int `json:"server_ids" binding:"required,min=1"`
}

// 批量健康检查响应
type BatchHealthCheckResponse struct {
	Results []HealthCheckResponse `json:"results"`
	Summary struct {
		Total     int `json:"total"`
		Healthy   int `json:"healthy"`
		Unhealthy int `json:"unhealthy"`
		Partial   int `json:"partial"`
	} `json:"summary"`
}
