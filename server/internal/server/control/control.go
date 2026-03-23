package control

import (
	"context"
	"io"
)

// Mode 控制模式
type Mode string

const (
	ModeLocal   Mode = "local"   // 单机模式：所有服务运行在本机
	ModeCluster Mode = "cluster" // 多机模式：服务分布在多台机器上
)

// ServiceName 支持的服务名
type ServiceName string

const (
	ServiceServer   ServiceName = "server"
	ServiceWuKongIM ServiceName = "wukongim"
	ServiceGost     ServiceName = "gost"
)

// SupportedServices 所有支持的服务列表
var SupportedServices = []ServiceName{ServiceServer, ServiceWuKongIM, ServiceGost}

// ServiceAction 服务操作
type ServiceAction string

const (
	ActionStart   ServiceAction = "start"
	ActionStop    ServiceAction = "stop"
	ActionRestart ServiceAction = "restart"
)

// --- Executor 接口：命令执行抽象 ---

// ExecResult 命令执行结果
type ExecResult struct {
	Output string
	Err    error
}

// Executor 命令执行器接口 —— 核心抽象层
// 单机模式直接在本地执行，多机模式通过 SSH 在远程执行
type Executor interface {
	// Execute 执行 shell 命令
	Execute(ctx context.Context, command string) ExecResult

	// UploadFile 上传文件
	UploadFile(ctx context.Context, remotePath string, reader io.Reader) error

	// Close 释放资源
	Close() error
}

// --- ServiceController 接口：服务管理抽象 ---

// ServiceStatus 服务状态
type ServiceStatus struct {
	ServiceName string `json:"service_name"`
	Status      string `json:"status"` // running / stopped / unknown
	Pid         int    `json:"pid"`
	Uptime      string `json:"uptime"`
	CPU         string `json:"cpu"`
	Memory      string `json:"memory"`
}

// ServiceActionResult 服务操作结果
type ServiceActionResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// ServiceLogs 服务日志
type ServiceLogs struct {
	Logs        string `json:"logs"`
	TotalLines  int    `json:"total_lines"`
	ServiceName string `json:"service_name"`
}

// ConfigFile 配置文件内容
type ConfigFile struct {
	ServiceName string `json:"service_name"`
	ConfigPath  string `json:"config_path"`
	Content     string `json:"content"`
}

// Endpoint 服务端点
type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ServerStats 服务器资源信息
type ServerStats struct {
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	MemoryTotal string `json:"memory_total"`
	DiskUsage   string `json:"disk_usage"`
	DiskTotal   string `json:"disk_total"`
	LoadAvg     string `json:"load_avg"`
}

// DockerContainer Docker 容器状态
type DockerContainer struct {
	ContainerId string `json:"container_id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	Ports       string `json:"ports"`
	Created     string `json:"created"`
	RunningFor  string `json:"running_for"`
	CPUPercent  string `json:"cpu_percent"`
	MemUsage    string `json:"mem_usage"`
	MemPercent  string `json:"mem_percent"`
}

// IController 统一控制器接口
// 同时兼容单机和多机模式
type IController interface {
	// --- 服务生命周期 ---

	// ServiceAction 执行服务操作（start/stop/restart）
	ServiceAction(ctx context.Context, service ServiceName, action ServiceAction) (ServiceActionResult, error)

	// GetServiceStatus 获取服务状态（传空则查询所有）
	GetServiceStatus(ctx context.Context, service ServiceName) ([]ServiceStatus, error)

	// GetServiceLogs 获取服务日志
	GetServiceLogs(ctx context.Context, service ServiceName, lines int) (ServiceLogs, error)

	// --- 配置管理 ---

	// GetConfigFile 获取服务配置文件
	GetConfigFile(ctx context.Context, service ServiceName) (ConfigFile, error)

	// UpdateConfigFile 更新服务配置文件（自动备份）
	UpdateConfigFile(ctx context.Context, service ServiceName, content string) (ConfigFile, error)

	// --- 文件部署 ---

	// DeployBinary 部署服务二进制文件（备份 + 原子替换）
	DeployBinary(ctx context.Context, service ServiceName, filename string, reader io.Reader) (string, error)

	// --- 监控 ---

	// GetServerStats 获取服务器资源使用
	GetServerStats(ctx context.Context) (ServerStats, error)

	// GetDockerContainers 获取 Docker 容器列表
	GetDockerContainers(ctx context.Context) ([]DockerContainer, error)

	// --- 服务发现 ---

	// GetEndpoints 获取指定服务的访问端点
	GetEndpoints(ctx context.Context, service ServiceName) ([]Endpoint, error)

	// --- 健康检查 ---

	// HealthCheck 检查所有服务健康状态
	HealthCheck(ctx context.Context) (map[ServiceName]ServiceStatus, error)

	// --- 元信息 ---

	// Mode 返回当前控制模式
	Mode() Mode

	// Close 释放资源
	Close() error
}
