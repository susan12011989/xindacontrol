package model

// ========== 批量服务操作 ==========

// BatchServiceActionReq 批量服务操作请求
type BatchServiceActionReq struct {
	ServerIds   []int  `json:"server_ids" binding:"required,min=1"`
	ServiceName string `json:"service_name" binding:"required,oneof=server wukongim gost"`
	Action      string `json:"action" binding:"required,oneof=start stop restart"`
	Parallel    bool   `json:"parallel"` // 是否并行执行，默认 false 顺序执行
}

// BatchServiceActionResp 批量服务操作响应
type BatchServiceActionResp struct {
	TotalCount   int                    `json:"total_count"`
	SuccessCount int                    `json:"success_count"`
	FailCount    int                    `json:"fail_count"`
	Results      []BatchServiceResult   `json:"results"`
}

// BatchServiceResult 单个服务器操作结果
type BatchServiceResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	ServerHost string `json:"server_host"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Output     string `json:"output,omitempty"`
}

// ========== 批量健康检查 ==========

// BatchHealthCheckReq 定义在 docker.go 中

// BatchHealthCheckResp 批量健康检查响应
type BatchHealthCheckResp struct {
	TotalCount    int                   `json:"total_count"`
	HealthyCount  int                   `json:"healthy_count"`
	UnhealthyCount int                  `json:"unhealthy_count"`
	PartialCount  int                   `json:"partial_count"`
	Results       []ServerHealthResult  `json:"results"`
}

// ServerHealthResult 单个服务器健康检查结果
type ServerHealthResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	ServerHost string `json:"server_host"`
	Status     string `json:"status"` // healthy/unhealthy/partial/error
	Message    string `json:"message,omitempty"`
	CheckTime  string `json:"check_time"`
}

// ========== 批量命令执行 ==========

// BatchCommandReq 批量命令执行请求
type BatchCommandReq struct {
	ServerIds []int  `json:"server_ids" binding:"required,min=1"`
	Command   string `json:"command" binding:"required"`
	Timeout   int    `json:"timeout"` // 超时时间（秒），默认 30
	Parallel  bool   `json:"parallel"`
}

// BatchCommandResp 批量命令执行响应
type BatchCommandResp struct {
	TotalCount   int                  `json:"total_count"`
	SuccessCount int                  `json:"success_count"`
	FailCount    int                  `json:"fail_count"`
	Results      []BatchCommandResult `json:"results"`
}

// BatchCommandResult 单个服务器命令执行结果
type BatchCommandResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	ServerHost string `json:"server_host"`
	Success    bool   `json:"success"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	Duration   int64  `json:"duration_ms"` // 执行时长（毫秒）
}

// ========== 日志查询 ==========

// ========== 同步群订阅者 ==========

// SyncSubscribersResp 同步群订阅者响应
type SyncSubscribersResp struct {
	TotalGroups  int    `json:"total_groups"`
	SyncedGroups int    `json:"synced_groups"`
	TotalMembers int    `json:"total_members"`
	FailedGroups int    `json:"failed_groups"`
	Message      string `json:"message"`
}

// ========== 日志查询 ==========

// LogQueryReq 日志查询请求
type LogQueryReq struct {
	ServerId      int    `json:"server_id" binding:"required"`
	QueryType     string `json:"query_type"`      // journalctl/docker/file，默认 journalctl
	ServiceName   string `json:"service_name"`    // server/wukongim/gost
	ContainerName string `json:"container_name"`  // Docker 容器名
	LogPath       string `json:"log_path"`        // 文件日志路径
	Lines         int    `json:"lines"`           // 行数，默认 100
	Since         string `json:"since"`           // 起始时间，如 "1h", "30m", "2024-01-01 00:00:00"
	Until         string `json:"until"`           // 结束时间
	Keyword       string `json:"keyword"`         // 关键字过滤
	Level         string `json:"level"`           // 日志级别过滤 error/warn/info
}

// LogQueryResp 日志查询响应
type LogQueryResp struct {
	Logs      string `json:"logs"`
	LineCount int    `json:"line_count"`
	Truncated bool   `json:"truncated"`
	Command   string `json:"command,omitempty"` // 实际执行的命令（调试用）
}
