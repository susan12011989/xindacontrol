package control

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

// LocalController 单机模式控制器
// 所有服务都运行在本机，直接通过本地命令管理
type LocalController struct {
	executor *LocalExecutor
}

// NewLocalController 创建单机控制器
func NewLocalController() *LocalController {
	return &LocalController{
		executor: NewLocalExecutor(),
	}
}

func (c *LocalController) Mode() Mode {
	return ModeLocal
}

func (c *LocalController) ServiceAction(ctx context.Context, svc ServiceName, action ServiceAction) (ServiceActionResult, error) {
	if err := ValidateServiceName(svc); err != nil {
		return ServiceActionResult{}, err
	}

	cmd := buildServiceActionCommand(svc, action)
	result := c.executor.Execute(ctx, cmd)

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
	return resp, nil
}

func (c *LocalController) GetServiceStatus(ctx context.Context, svc ServiceName) ([]ServiceStatus, error) {
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
		result := c.executor.Execute(ctx, cmd)
		status := parseServiceStatusOutput(s, result.Output)
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (c *LocalController) GetServiceLogs(ctx context.Context, svc ServiceName, lines int) (ServiceLogs, error) {
	if err := ValidateServiceName(svc); err != nil {
		return ServiceLogs{}, err
	}
	if lines == 0 {
		lines = 100
	}

	systemdName := serviceSystemdNames[svc]
	dockerName := serviceDockerNames[svc]

	// 优先 journalctl，回退 docker logs
	cmd := fmt.Sprintf(`
journalctl -u %s -n %d --no-pager 2>/dev/null
if [ $? -ne 0 ]; then
    docker logs --tail %d %s 2>&1 || echo "无法获取日志"
fi
`, systemdName, lines, lines, dockerName)

	result := c.executor.Execute(ctx, cmd)
	output := result.Output
	return ServiceLogs{
		Logs:        output,
		TotalLines:  len(strings.Split(output, "\n")),
		ServiceName: string(svc),
	}, nil
}

func (c *LocalController) GetConfigFile(ctx context.Context, svc ServiceName) (ConfigFile, error) {
	configPath, ok := serviceConfigPaths[svc]
	if !ok {
		return ConfigFile{}, fmt.Errorf("服务 %s 没有配置文件", svc)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ConfigFile{ServiceName: string(svc), ConfigPath: configPath, Content: ""}, nil
		}
		return ConfigFile{}, fmt.Errorf("读取配置文件失败: %v", err)
	}

	return ConfigFile{
		ServiceName: string(svc),
		ConfigPath:  configPath,
		Content:     string(content),
	}, nil
}

func (c *LocalController) UpdateConfigFile(ctx context.Context, svc ServiceName, content string) (ConfigFile, error) {
	configPath, ok := serviceConfigPaths[svc]
	if !ok {
		return ConfigFile{}, fmt.Errorf("服务 %s 没有配置文件", svc)
	}

	// 备份原配置
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", configPath, ts)
	if _, err := os.Stat(configPath); err == nil {
		// 文件存在，备份
		data, _ := os.ReadFile(configPath)
		if data != nil {
			os.WriteFile(backupPath, data, 0644)
		}
	}

	// 确保目录存在
	dir := path.Dir(configPath)
	os.MkdirAll(dir, 0755)

	// 写入新配置
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return ConfigFile{}, fmt.Errorf("写入配置文件失败: %v", err)
	}

	return ConfigFile{
		ServiceName: string(svc),
		ConfigPath:  configPath,
		Content:     content,
	}, nil
}

func (c *LocalController) DeployBinary(ctx context.Context, svc ServiceName, filename string, reader io.Reader) (string, error) {
	uploadPath, ok := serviceUploadPaths[svc]
	if !ok {
		return "", fmt.Errorf("服务 %s 不支持部署", svc)
	}
	binaryName := serviceBinaryNames[svc]
	if binaryName == "" {
		binaryName = filename
	}
	remotePath := path.Join(uploadPath, binaryName)

	// 确保目录存在
	os.MkdirAll(uploadPath, 0755)

	// 备份
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", remotePath, ts)
	if _, err := os.Stat(remotePath); err == nil {
		os.Rename(remotePath, backupPath)
	}

	// 写入临时文件再原子替换
	tmpPath := fmt.Sprintf("%s.%s.tmp", remotePath, ts)
	if err := c.executor.UploadFile(ctx, tmpPath, reader); err != nil {
		return "", fmt.Errorf("写入文件失败: %v", err)
	}
	if err := os.Rename(tmpPath, remotePath); err != nil {
		return "", fmt.Errorf("替换文件失败: %v", err)
	}
	os.Chmod(remotePath, 0755)

	return remotePath, nil
}

func (c *LocalController) GetServerStats(ctx context.Context) (ServerStats, error) {
	cmd := buildServerStatsCommand()
	result := c.executor.Execute(ctx, cmd)
	if result.Err != nil {
		return ServerStats{}, fmt.Errorf("获取服务器资源失败: %v", result.Err)
	}
	return parseServerStats(result.Output), nil
}

func (c *LocalController) GetDockerContainers(ctx context.Context) ([]DockerContainer, error) {
	listResult := c.executor.Execute(ctx, buildDockerContainersCommand())
	statsResult := c.executor.Execute(ctx, buildDockerStatsCommand())
	return parseDockerContainers(listResult.Output, statsResult.Output), nil
}

func (c *LocalController) GetEndpoints(_ context.Context, svc ServiceName) ([]Endpoint, error) {
	if err := ValidateServiceName(svc); err != nil {
		return nil, err
	}
	port := serviceDefaultPorts[svc]
	return []Endpoint{{Host: "127.0.0.1", Port: port}}, nil
}

func (c *LocalController) HealthCheck(ctx context.Context) (map[ServiceName]ServiceStatus, error) {
	result := make(map[ServiceName]ServiceStatus)
	for _, svc := range SupportedServices {
		statuses, err := c.GetServiceStatus(ctx, svc)
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

func (c *LocalController) Close() error {
	return c.executor.Close()
}
