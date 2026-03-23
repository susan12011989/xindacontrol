package control

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ClusterController 多机模式控制器
// 通过 SSH 连接池管理远程服务器上的服务
type ClusterController struct {
	// 无状态，每次操作通过 serverId 获取对应 executor
}

// NewClusterController 创建多机控制器
func NewClusterController() *ClusterController {
	return &ClusterController{}
}

func (c *ClusterController) Mode() Mode {
	return ModeCluster
}

// getExecutor 根据 serverId 获取远程执行器
func (c *ClusterController) getExecutor(serverId int) (*RemoteExecutor, error) {
	client, err := getSSHClient(serverId)
	if err != nil {
		return nil, err
	}
	return NewRemoteExecutor(client), nil
}

// --- 带 serverId 的多机操作方法 ---

// ServiceActionOnServer 在指定服务器上执行服务操作
func (c *ClusterController) ServiceActionOnServer(ctx context.Context, serverId int, svc ServiceName, action ServiceAction, operator string) (ServiceActionResult, error) {
	if err := ValidateServiceName(svc); err != nil {
		return ServiceActionResult{}, err
	}

	executor, err := c.getExecutor(serverId)
	if err != nil {
		return ServiceActionResult{}, err
	}

	cmd := buildServiceActionCommand(svc, action)
	result := executor.Execute(ctx, cmd)

	resp := ServiceActionResult{
		Output: result.Output,
	}
	if result.Err != nil {
		resp.Success = false
		resp.ErrorMsg = result.Err.Error()
		resp.Message = fmt.Sprintf("%s %s 失败", action, svc)
	} else {
		resp.Success = true
		resp.Message = fmt.Sprintf("%s %s 成功", action, svc)
	}

	// 记录操作历史
	history := entity.DeployHistory{
		ServerId:    serverId,
		Action:      string(action),
		ServiceName: string(svc),
		Operator:    operator,
		Status:      1,
		Output:      result.Output,
		CreatedAt:   time.Now(),
	}
	if result.Err != nil {
		history.Status = 2
		history.ErrorMsg = result.Err.Error()
	}
	dbs.DBAdmin.Insert(&history)

	return resp, nil
}

// GetServiceStatusOnServer 获取指定服务器上的服务状态
func (c *ClusterController) GetServiceStatusOnServer(ctx context.Context, serverId int, svc ServiceName) ([]ServiceStatus, error) {
	executor, err := c.getExecutor(serverId)
	if err != nil {
		return nil, err
	}

	services := SupportedServices
	if svc != "" {
		if err := ValidateServiceName(svc); err != nil {
			return nil, err
		}
		services = []ServiceName{svc}
	}

	var statuses []ServiceStatus
	for _, s := range services {
		cmd := buildServiceStatusCommand(s)
		result := executor.Execute(ctx, cmd)
		status := parseServiceStatusOutput(s, result.Output)
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// GetServiceLogsOnServer 获取指定服务器上的服务日志
func (c *ClusterController) GetServiceLogsOnServer(ctx context.Context, serverId int, svc ServiceName, lines int) (ServiceLogs, error) {
	if err := ValidateServiceName(svc); err != nil {
		return ServiceLogs{}, err
	}
	if lines == 0 {
		lines = 100
	}

	executor, err := c.getExecutor(serverId)
	if err != nil {
		return ServiceLogs{}, err
	}

	systemdName := serviceSystemdNames[svc]
	dockerName := serviceDockerNames[svc]
	cmd := fmt.Sprintf(`
journalctl -u %s -n %d --no-pager 2>/dev/null
if [ $? -ne 0 ]; then
    docker logs --tail %d %s 2>&1 || echo "无法获取日志"
fi
`, systemdName, lines, lines, dockerName)

	result := executor.Execute(ctx, cmd)
	output := result.Output
	return ServiceLogs{
		Logs:        output,
		TotalLines:  len(strings.Split(output, "\n")),
		ServiceName: string(svc),
	}, nil
}

// GetConfigFileOnServer 获取指定服务器上的配置文件
func (c *ClusterController) GetConfigFileOnServer(ctx context.Context, serverId int, svc ServiceName) (ConfigFile, error) {
	configPath, ok := serviceConfigPaths[svc]
	if !ok {
		return ConfigFile{}, fmt.Errorf("服务 %s 没有配置文件", svc)
	}

	executor, err := c.getExecutor(serverId)
	if err != nil {
		return ConfigFile{}, err
	}

	// 检查文件是否存在
	checkResult := executor.Execute(ctx, fmt.Sprintf("sudo test -f '%s' && echo 'exists'", configPath))
	if strings.TrimSpace(checkResult.Output) != "exists" {
		return ConfigFile{ServiceName: string(svc), ConfigPath: configPath, Content: ""}, nil
	}

	result := executor.Execute(ctx, fmt.Sprintf("sudo cat '%s'", configPath))
	if result.Err != nil {
		return ConfigFile{}, fmt.Errorf("读取配置文件失败: %v", result.Err)
	}

	return ConfigFile{
		ServiceName: string(svc),
		ConfigPath:  configPath,
		Content:     result.Output,
	}, nil
}

// UpdateConfigFileOnServer 更新指定服务器上的配置文件
func (c *ClusterController) UpdateConfigFileOnServer(ctx context.Context, serverId int, svc ServiceName, content string) (ConfigFile, error) {
	configPath, ok := serviceConfigPaths[svc]
	if !ok {
		return ConfigFile{}, fmt.Errorf("服务 %s 没有配置文件", svc)
	}

	executor, err := c.getExecutor(serverId)
	if err != nil {
		return ConfigFile{}, err
	}

	// 备份原配置
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", configPath, ts)
	executor.Execute(ctx, fmt.Sprintf("sudo cp -f '%s' '%s' 2>/dev/null", configPath, backupPath))

	// 确保目录存在
	dir := path.Dir(configPath)
	executor.Execute(ctx, fmt.Sprintf("sudo mkdir -p '%s'", dir))

	// 写入新配置
	escapedContent := strings.ReplaceAll(content, "'", "'\"'\"'")
	result := executor.Execute(ctx, fmt.Sprintf("echo '%s' | sudo tee '%s' > /dev/null", escapedContent, configPath))
	if result.Err != nil {
		return ConfigFile{}, fmt.Errorf("写入配置文件失败: %v", result.Err)
	}

	return ConfigFile{
		ServiceName: string(svc),
		ConfigPath:  configPath,
		Content:     content,
	}, nil
}

// DeployBinaryToServer 部署二进制文件到指定服务器
func (c *ClusterController) DeployBinaryToServer(ctx context.Context, serverId int, svc ServiceName, filename string, reader io.Reader) (string, error) {
	uploadPath, ok := serviceUploadPaths[svc]
	if !ok {
		return "", fmt.Errorf("服务 %s 不支持部署", svc)
	}
	binaryName := serviceBinaryNames[svc]
	if binaryName == "" {
		binaryName = filename
	}
	remotePath := path.Join(uploadPath, binaryName)

	executor, err := c.getExecutor(serverId)
	if err != nil {
		return "", err
	}

	// 确保目录存在
	executor.Execute(ctx, fmt.Sprintf("mkdir -p %s", uploadPath))

	// 备份
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", remotePath, ts)
	executor.Execute(ctx, fmt.Sprintf("if [ -f '%s' ]; then cp -f '%s' '%s'; fi", remotePath, remotePath, backupPath))

	// 上传到临时文件再原子替换
	tmpPath := fmt.Sprintf("%s.%s.tmp", remotePath, ts)
	if err := executor.UploadFile(ctx, tmpPath, reader); err != nil {
		return "", fmt.Errorf("上传文件失败: %v", err)
	}

	result := executor.Execute(ctx, fmt.Sprintf("mv -f '%s' '%s' && chmod +x '%s'", tmpPath, remotePath, remotePath))
	if result.Err != nil {
		return "", fmt.Errorf("替换文件失败: %v", result.Err)
	}

	return remotePath, nil
}

// GetServerStatsOnServer 获取指定服务器的资源使用
func (c *ClusterController) GetServerStatsOnServer(ctx context.Context, serverId int) (ServerStats, error) {
	executor, err := c.getExecutor(serverId)
	if err != nil {
		return ServerStats{}, err
	}

	result := executor.Execute(ctx, buildServerStatsCommand())
	if result.Err != nil {
		return ServerStats{}, fmt.Errorf("获取服务器资源失败: %v", result.Err)
	}
	return parseServerStats(result.Output), nil
}

// GetDockerContainersOnServer 获取指定服务器的 Docker 容器列表
func (c *ClusterController) GetDockerContainersOnServer(ctx context.Context, serverId int) ([]DockerContainer, error) {
	executor, err := c.getExecutor(serverId)
	if err != nil {
		return nil, err
	}

	listResult := executor.Execute(ctx, buildDockerContainersCommand())
	statsResult := executor.Execute(ctx, buildDockerStatsCommand())
	return parseDockerContainers(listResult.Output, statsResult.Output), nil
}

// GetEndpointsOnServer 获取指定服务器上的服务端点
func (c *ClusterController) GetEndpointsOnServer(_ context.Context, serverId int, svc ServiceName) ([]Endpoint, error) {
	if err := ValidateServiceName(svc); err != nil {
		return nil, err
	}

	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("服务器 %d 不存在", serverId)
	}

	port := serviceDefaultPorts[svc]
	return []Endpoint{{Host: server.Host, Port: port}}, nil
}

// HealthCheckOnServer 检查指定服务器上所有服务的健康状态
func (c *ClusterController) HealthCheckOnServer(ctx context.Context, serverId int) (map[ServiceName]ServiceStatus, error) {
	result := make(map[ServiceName]ServiceStatus)
	for _, svc := range SupportedServices {
		statuses, err := c.GetServiceStatusOnServer(ctx, serverId, svc)
		if err != nil {
			result[svc] = ServiceStatus{ServiceName: string(svc), Status: "unknown"}
			continue
		}
		if len(statuses) > 0 {
			result[svc] = statuses[0]
		}
	}
	return result, nil
}

// HealthCheckAllServers 健康检查所有活跃服务器
func (c *ClusterController) HealthCheckAllServers(ctx context.Context) (map[int]map[ServiceName]ServiceStatus, error) {
	var servers []entity.Servers
	if err := dbs.DBAdmin.Where("status = 1").Find(&servers); err != nil {
		return nil, err
	}

	allResults := make(map[int]map[ServiceName]ServiceStatus)
	for _, server := range servers {
		result, err := c.HealthCheckOnServer(ctx, server.Id)
		if err != nil {
			logx.Errorf("health check server %d failed: %v", server.Id, err)
			continue
		}
		allResults[server.Id] = result
	}
	return allResults, nil
}

// --- IController 接口实现（代理到第一台活跃服务器，用于兼容统一接口） ---

func (c *ClusterController) ServiceAction(ctx context.Context, svc ServiceName, action ServiceAction) (ServiceActionResult, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return ServiceActionResult{}, err
	}
	return c.ServiceActionOnServer(ctx, serverId, svc, action, "system")
}

func (c *ClusterController) GetServiceStatus(ctx context.Context, svc ServiceName) ([]ServiceStatus, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return c.GetServiceStatusOnServer(ctx, serverId, svc)
}

func (c *ClusterController) GetServiceLogs(ctx context.Context, svc ServiceName, lines int) (ServiceLogs, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return ServiceLogs{}, err
	}
	return c.GetServiceLogsOnServer(ctx, serverId, svc, lines)
}

func (c *ClusterController) GetConfigFile(ctx context.Context, svc ServiceName) (ConfigFile, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return ConfigFile{}, err
	}
	return c.GetConfigFileOnServer(ctx, serverId, svc)
}

func (c *ClusterController) UpdateConfigFile(ctx context.Context, svc ServiceName, content string) (ConfigFile, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return ConfigFile{}, err
	}
	return c.UpdateConfigFileOnServer(ctx, serverId, svc, content)
}

func (c *ClusterController) DeployBinary(ctx context.Context, svc ServiceName, filename string, reader io.Reader) (string, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return "", err
	}
	return c.DeployBinaryToServer(ctx, serverId, svc, filename, reader)
}

func (c *ClusterController) GetServerStats(ctx context.Context) (ServerStats, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return ServerStats{}, err
	}
	return c.GetServerStatsOnServer(ctx, serverId)
}

func (c *ClusterController) GetDockerContainers(ctx context.Context) ([]DockerContainer, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return c.GetDockerContainersOnServer(ctx, serverId)
}

func (c *ClusterController) GetEndpoints(ctx context.Context, svc ServiceName) ([]Endpoint, error) {
	// 多机模式：返回所有活跃服务器的端点
	var servers []entity.Servers
	if err := dbs.DBAdmin.Where("status = 1").Find(&servers); err != nil {
		return nil, err
	}
	port := serviceDefaultPorts[svc]
	var endpoints []Endpoint
	for _, s := range servers {
		endpoints = append(endpoints, Endpoint{Host: s.Host, Port: port})
	}
	return endpoints, nil
}

func (c *ClusterController) HealthCheck(ctx context.Context) (map[ServiceName]ServiceStatus, error) {
	serverId, err := c.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return c.HealthCheckOnServer(ctx, serverId)
}

func (c *ClusterController) Close() error {
	return nil
}

// --- 内部辅助 ---

// firstActiveServerId 获取第一台活跃服务器 ID（IController 默认目标）
func (c *ClusterController) firstActiveServerId() (int, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("status = 1").Asc("id").Get(&server)
	if err != nil {
		return 0, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return 0, fmt.Errorf("没有可用的活跃服务器")
	}
	return server.Id, nil
}

// getServerEntity 根据 ID 查询服务器实体
func getServerEntity(serverId int) (*entity.Servers, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在: %d", serverId)
	}
	return &server, nil
}

// getSSHClient 获取 SSH 客户端（复用连接池）
func getSSHClient(serverId int) (*utils.PooledSSHClient, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在")
	}
	if server.Status != 1 {
		return nil, fmt.Errorf("服务器已禁用")
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

// DistributeBinary 批量分发二进制到多台服务器
func (c *ClusterController) DistributeBinary(ctx context.Context, svc ServiceName, targetServerIds []int, restartAfter bool, operator string) ([]DistributeResult, error) {
	uploadPath, ok := serviceUploadPaths[svc]
	if !ok {
		return nil, fmt.Errorf("服务 %s 不支持分发", svc)
	}
	binaryName := serviceBinaryNames[svc]
	if binaryName == "" {
		binaryName = string(svc)
	}

	// 本地暂存文件路径
	localFilePath := path.Join("/tmp/deploy_uploads", string(svc))
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("本地文件 %s 不存在，请先上传程序", localFilePath)
	}

	remotePath := path.Join(uploadPath, binaryName)

	var targetServers []entity.Servers
	if err := dbs.DBAdmin.In("id", targetServerIds).Find(&targetServers); err != nil {
		return nil, fmt.Errorf("查询目标服务器失败: %v", err)
	}

	var results []DistributeResult
	for _, target := range targetServers {
		dr := DistributeResult{
			ServerId:   target.Id,
			ServerName: target.Name,
		}

		executor, err := c.getExecutor(target.Id)
		if err != nil {
			dr.Success = false
			dr.Message = fmt.Sprintf("连接服务器失败: %v", err)
			results = append(results, dr)
			continue
		}

		// 确保远程目录
		executor.Execute(ctx, fmt.Sprintf("mkdir -p %s", uploadPath))

		// 备份
		ts := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.%s.bak", remotePath, ts)
		executor.Execute(ctx, fmt.Sprintf("if [ -f '%s' ]; then cp -f '%s' '%s'; fi", remotePath, remotePath, backupPath))

		// 上传
		localFile, err := os.Open(localFilePath)
		if err != nil {
			dr.Success = false
			dr.Message = fmt.Sprintf("读取本地文件失败: %v", err)
			results = append(results, dr)
			continue
		}

		tmpPath := fmt.Sprintf("%s.%s.tmp", remotePath, ts)
		uploadErr := executor.UploadFile(ctx, tmpPath, localFile)
		localFile.Close()

		if uploadErr != nil {
			dr.Success = false
			dr.Message = fmt.Sprintf("上传文件失败: %v", uploadErr)
			results = append(results, dr)
			continue
		}

		// 原子替换 + 权限
		result := executor.Execute(ctx, fmt.Sprintf("mv -f '%s' '%s' && chmod +x '%s'", tmpPath, remotePath, remotePath))
		if result.Err != nil {
			dr.Success = false
			dr.Message = fmt.Sprintf("替换文件失败: %v", result.Err)
			results = append(results, dr)
			continue
		}

		// 可选重启
		if restartAfter {
			restartResult := executor.Execute(ctx, buildServiceActionCommand(svc, ActionRestart))
			if restartResult.Err != nil {
				dr.Success = true
				dr.Message = fmt.Sprintf("传输成功，但重启失败: %v", restartResult.Err)
			} else {
				dr.Success = true
				dr.Message = "传输成功，服务已重启"
			}
		} else {
			dr.Success = true
			dr.Message = "传输成功"
		}

		results = append(results, dr)

		// 记录历史
		history := entity.DeployHistory{
			ServerId:    target.Id,
			Action:      "distribute",
			ServiceName: string(svc),
			Operator:    operator,
			Status:      1,
			Output:      dr.Message,
			CreatedAt:   time.Now(),
		}
		if !dr.Success {
			history.Status = 2
			history.ErrorMsg = dr.Message
		}
		dbs.DBAdmin.Insert(&history)
	}

	return results, nil
}

// DistributeResult 单台服务器分发结果
type DistributeResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}
