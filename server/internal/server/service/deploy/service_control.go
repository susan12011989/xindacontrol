package deploy

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strconv"
	"strings"
	"time"
)

// 服务名到 systemctl 服务名的映射
var serviceSystemdNames = map[string]string{
	"server":   "server.service",
	"wukongim": "wukongim.service",
	"gost":     "gost.service",
}

// 服务配置文件路径映射
var serviceConfigPaths = map[string]string{
	"server":   "/root/server/configs/tsdd.yaml",
	"wukongim": "/root/wukongim/wk.yaml",
	"gost":     "/root/gost/gost.yaml",
}

// ServiceAction 执行服务操作（start/stop/restart）
func ServiceAction(req model.ServiceActionReq, operator string) (model.ServiceActionResp, error) {
	var resp model.ServiceActionResp

	// 验证服务名
	systemdName, ok := serviceSystemdNames[req.ServiceName]
	if !ok {
		return resp, fmt.Errorf("不支持的服务: %s，仅支持 server/wukongim/gost", req.ServiceName)
	}

	// 获取 SSH 客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 构建 systemctl 命令
	cmd := fmt.Sprintf("systemctl %s %s", req.Action, systemdName)
	output, execErr := client.ExecuteCommand(cmd)

	// 构造响应
	resp.Output = output
	if execErr != nil {
		resp.Success = false
		resp.ErrorMsg = execErr.Error()
		resp.Message = fmt.Sprintf("%s %s 失败", req.Action, req.ServiceName)
	} else {
		resp.Success = true
		resp.Message = fmt.Sprintf("%s %s 成功", req.Action, req.ServiceName)
	}

	// 记录操作历史
	history := entity.DeployHistory{
		ServerId:    req.ServerId,
		Action:      req.Action,
		ServiceName: req.ServiceName,
		Operator:    operator,
		Status:      1,
		Output:      output,
		CreatedAt:   time.Now(),
	}
	if execErr != nil {
		history.Status = 2
		history.ErrorMsg = execErr.Error()
	}
	dbs.DBAdmin.Insert(&history)

	return resp, nil
}

// GetServiceStatus 获取服务状态
func GetServiceStatus(req model.ServiceStatusReq) (model.ServiceStatusListResp, error) {
	var resp model.ServiceStatusListResp

	// 获取 SSH 客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 确定要查询的服务列表
	services := model.SupportedServices
	if req.ServiceName != "" {
		if _, ok := serviceSystemdNames[req.ServiceName]; !ok {
			return resp, fmt.Errorf("不支持的服务: %s", req.ServiceName)
		}
		services = []string{req.ServiceName}
	}

	// 查询每个服务的状态
	for _, svc := range services {
		systemdName := serviceSystemdNames[svc]
		status := queryServiceStatus(client.SSHClient, systemdName)
		status.ServiceName = svc
		resp.Services = append(resp.Services, status)
	}

	return resp, nil
}

// queryServiceStatus 查询单个服务的状态
func queryServiceStatus(client *utils.SSHClient, systemdName string) model.ServiceStatusResp {
	var status model.ServiceStatusResp

	// 检查服务是否运行
	cmd := fmt.Sprintf("systemctl is-active %s 2>/dev/null || echo 'inactive'", systemdName)
	output, _ := client.ExecuteCommand(cmd)
	output = strings.TrimSpace(output)

	if output == "active" {
		status.Status = "running"

		// 获取 PID
		pidCmd := fmt.Sprintf("systemctl show %s --property=MainPID --value", systemdName)
		pidOutput, _ := client.ExecuteCommand(pidCmd)
		pidOutput = strings.TrimSpace(pidOutput)
		if pid, err := strconv.Atoi(pidOutput); err == nil && pid > 0 {
			status.Pid = pid

			// 获取进程资源使用
			statsCmd := fmt.Sprintf("ps -p %d -o %%cpu=,%%mem=,rss=,etime= 2>/dev/null", pid)
			statsOutput, _ := client.ExecuteCommand(statsCmd)
			statsOutput = strings.TrimSpace(statsOutput)
			if statsOutput != "" {
				parts := strings.Fields(statsOutput)
				if len(parts) >= 4 {
					status.CPU = parts[0] + "%"
					if rss, err := strconv.ParseFloat(parts[2], 64); err == nil {
						status.Memory = formatMemorySize(rss)
					}
					status.Uptime = parts[3]
				}
			}
		}
	} else {
		status.Status = "stopped"
	}

	return status
}

// formatMemorySize 格式化内存（KB 转可读格式）
func formatMemorySize(kb float64) string {
	if kb < 1024 {
		return fmt.Sprintf("%.0fKB", kb)
	} else if kb < 1024*1024 {
		return fmt.Sprintf("%.1fMB", kb/1024)
	}
	return fmt.Sprintf("%.2fGB", kb/1024/1024)
}

// GetServiceLogs 获取服务日志（使用 journalctl）
func GetServiceLogs(req model.ServiceLogsReq) (model.ServiceLogsResp, error) {
	var resp model.ServiceLogsResp

	systemdName, ok := serviceSystemdNames[req.ServiceName]
	if !ok {
		return resp, fmt.Errorf("不支持的服务: %s", req.ServiceName)
	}

	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	lines := req.Lines
	if lines == 0 {
		lines = 100
	}

	cmd := fmt.Sprintf("journalctl -u %s -n %d --no-pager", systemdName, lines)
	output, err := client.ExecuteCommand(cmd)
	if err != nil {
		return resp, fmt.Errorf("读取日志失败: %v", err)
	}

	resp.Logs = output
	resp.TotalLines = len(strings.Split(output, "\n"))
	resp.ServiceName = req.ServiceName

	return resp, nil
}

// UploadServiceFile 上传服务文件（仅 server 和 wukongim）
func UploadServiceFile(serverId int, serviceName string, filename string, reader io.Reader) (string, error) {
	// 验证服务名和获取上传路径
	uploadPath, ok := model.ServiceUploadPaths[serviceName]
	if !ok {
		return "", fmt.Errorf("服务 %s 不支持上传", serviceName)
	}

	// 获取 SSH 客户端
	client, err := GetSSHClient(serverId)
	if err != nil {
		return "", err
	}

	// 获取实际的二进制文件名（与 systemd 配置中的 ExecStart 一致）
	binaryName, ok := model.ServiceBinaryNames[serviceName]
	if !ok {
		binaryName = filename // 兜底使用传入的文件名
	}

	// 构建远端路径（使用实际的二进制文件名）
	remotePath := path.Join(uploadPath, binaryName)

	// 确保目录存在
	if _, err = client.ExecuteCommand(fmt.Sprintf("mkdir -p %s", uploadPath)); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	// 备份原文件
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", remotePath, ts)
	client.ExecuteCommand(fmt.Sprintf("if [ -f '%s' ]; then cp -f '%s' '%s'; fi", remotePath, remotePath, backupPath))

	// 上传到临时文件再原子替换
	tmpPath := fmt.Sprintf("%s.%s.tmp", remotePath, ts)
	if err = client.UploadFile(tmpPath, reader); err != nil {
		return "", fmt.Errorf("上传文件失败: %v", err)
	}

	// 原子替换
	if _, err = client.ExecuteCommand(fmt.Sprintf("mv -f '%s' '%s'", tmpPath, remotePath)); err != nil {
		return "", fmt.Errorf("替换文件失败: %v", err)
	}

	// 设置可执行权限
	client.ExecuteCommand(fmt.Sprintf("chmod +x '%s'", remotePath))

	return remotePath, nil
}

// GetConfigFile 获取配置文件内容
func GetConfigFile(serverId int, serviceName string) (model.ConfigFileResp, error) {
	var resp model.ConfigFileResp

	configPath, ok := serviceConfigPaths[serviceName]
	if !ok {
		return resp, fmt.Errorf("服务 %s 没有配置文件", serviceName)
	}

	client, err := GetSSHClient(serverId)
	if err != nil {
		return resp, err
	}

	cmd := fmt.Sprintf("cat '%s'", configPath)
	content, err := client.ExecuteCommand(cmd)
	if err != nil {
		return resp, fmt.Errorf("读取配置文件失败: %v", err)
	}

	resp.ServiceName = serviceName
	resp.ConfigPath = configPath
	resp.Content = content

	return resp, nil
}

// UpdateConfigFile 更新配置文件内容（自动备份）
func UpdateConfigFile(serverId int, serviceName string, content string) (model.ConfigFileResp, error) {
	var resp model.ConfigFileResp

	configPath, ok := serviceConfigPaths[serviceName]
	if !ok {
		return resp, fmt.Errorf("服务 %s 没有配置文件", serviceName)
	}

	client, err := GetSSHClient(serverId)
	if err != nil {
		return resp, err
	}

	// 备份原配置
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", configPath, ts)
	client.ExecuteCommand(fmt.Sprintf("cp -f '%s' '%s' 2>/dev/null", configPath, backupPath))

	// 写入新配置
	escapedContent := strings.ReplaceAll(content, "'", "'\"'\"'")
	cmd := fmt.Sprintf("echo '%s' > '%s'", escapedContent, configPath)
	if _, err = client.ExecuteCommand(cmd); err != nil {
		return resp, fmt.Errorf("写入配置文件失败: %v", err)
	}

	resp.ServiceName = serviceName
	resp.ConfigPath = configPath
	resp.Content = content

	return resp, nil
}

// UploadToLocal 上传文件到本地临时目录（用于批量分发）
func UploadToLocal(serviceName string, filename string, reader io.Reader) (string, error) {
	// 验证服务名
	if serviceName != "server" && serviceName != "wukongim" {
		return "", fmt.Errorf("服务 %s 不支持上传", serviceName)
	}

	// 确保本地上传目录存在
	uploadDir := model.LocalUploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("创建上传目录失败: %v", err)
	}

	// 本地文件路径（使用服务名作为文件名）
	localPath := path.Join(uploadDir, serviceName)

	// 备份原文件
	if _, err := os.Stat(localPath); err == nil {
		ts := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.%s.bak", localPath, ts)
		os.Rename(localPath, backupPath)
	}

	// 写入文件
	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("写入文件失败: %v", err)
	}

	// 设置可执行权限
	os.Chmod(localPath, 0755)

	return localPath, nil
}

// DistributeFile 批量分发文件
// 从本地服务器通过 SCP 传输文件到多个目标服务器
func DistributeFile(req model.DistributeFileReq, operator string) (model.DistributeFileResp, error) {
	var resp model.DistributeFileResp

	// 验证服务名
	remoteUploadPath, ok := model.ServiceUploadPaths[req.ServiceName]
	if !ok {
		return resp, fmt.Errorf("服务 %s 不支持分发", req.ServiceName)
	}

	// 获取实际的二进制文件名（与 systemd 配置中的 ExecStart 一致）
	binaryName, ok := model.ServiceBinaryNames[req.ServiceName]
	if !ok {
		binaryName = req.ServiceName // 兜底使用服务名
	}

	// 本地文件路径（使用服务名存储）
	localFilePath := path.Join(model.LocalUploadDir, req.ServiceName)

	// 检查本地文件是否存在
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		return resp, fmt.Errorf("本地文件 %s 不存在，请先上传程序", localFilePath)
	}

	// 远程目标路径（使用实际的二进制文件名）
	remoteFilePath := path.Join(remoteUploadPath, binaryName)

	// 获取目标服务器列表
	var targetServers []entity.Servers
	err := dbs.DBAdmin.In("id", req.TargetServerIds).Find(&targetServers)
	if err != nil {
		return resp, fmt.Errorf("查询目标服务器失败: %v", err)
	}

	resp.TotalCount = len(targetServers)
	resp.Results = make([]model.DistributeResult, 0, len(targetServers))

	// 逐个分发到目标服务器
	for _, target := range targetServers {
		result := model.DistributeResult{
			ServerId:   target.Id,
			ServerName: target.Name,
		}

		// 获取目标服务器 SSH 客户端
		targetClient, err := GetSSHClient(target.Id)
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("连接目标服务器失败: %v", err)
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}

		// 确保远程目录存在
		targetClient.ExecuteCommand(fmt.Sprintf("mkdir -p %s", remoteUploadPath))

		// 备份远程原文件
		ts := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.%s.bak", remoteFilePath, ts)
		targetClient.ExecuteCommand(fmt.Sprintf("if [ -f '%s' ]; then cp -f '%s' '%s'; fi", remoteFilePath, remoteFilePath, backupPath))

		// 读取本地文件并上传
		localFile, err := os.Open(localFilePath)
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("读取本地文件失败: %v", err)
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}

		// 上传到临时文件
		tmpPath := fmt.Sprintf("%s.%s.tmp", remoteFilePath, ts)
		uploadErr := targetClient.SSHClient.UploadFile(tmpPath, localFile)
		localFile.Close()

		if uploadErr != nil {
			result.Success = false
			result.Message = fmt.Sprintf("上传文件失败: %v", uploadErr)
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}

		// 原子替换
		_, mvErr := targetClient.ExecuteCommand(fmt.Sprintf("mv -f '%s' '%s'", tmpPath, remoteFilePath))
		if mvErr != nil {
			result.Success = false
			result.Message = fmt.Sprintf("替换文件失败: %v", mvErr)
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}

		// 设置可执行权限
		targetClient.ExecuteCommand(fmt.Sprintf("chmod +x '%s'", remoteFilePath))

		// 如果需要重启服务
		if req.RestartAfter {
			systemdName := serviceSystemdNames[req.ServiceName]
			restartCmd := fmt.Sprintf("systemctl restart %s", systemdName)
			_, restartErr := targetClient.ExecuteCommand(restartCmd)
			if restartErr != nil {
				result.Success = true
				result.Message = fmt.Sprintf("传输成功，但重启失败: %v", restartErr)
			} else {
				result.Success = true
				result.Message = "传输成功，服务已重启"
			}
		} else {
			result.Success = true
			result.Message = "传输成功"
		}

		resp.Results = append(resp.Results, result)
		resp.SuccessCount++

		// 记录操作历史
		history := entity.DeployHistory{
			ServerId:    target.Id,
			Action:      "distribute",
			ServiceName: req.ServiceName,
			Operator:    operator,
			Status:      1,
			Output:      result.Message,
			CreatedAt:   time.Now(),
		}
		if !result.Success {
			history.Status = 2
			history.ErrorMsg = result.Message
		}
		dbs.DBAdmin.Insert(&history)
	}

	return resp, nil
}

// GetDockerContainers 获取 Docker 容器状态
func GetDockerContainers(serverId int) (model.DockerContainersResp, error) {
	var resp model.DockerContainersResp

	client, err := GetSSHClient(serverId)
	if err != nil {
		return resp, err
	}

	// 使用 docker ps -a 获取所有容器，格式化输出
	// 格式: ID|Name|Image|Status|Ports|Created|RunningFor
	cmd := `docker ps -a --format "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}|{{.CreatedAt}}|{{.RunningFor}}" 2>/dev/null || echo ""`
	output, err := client.ExecuteCommand(cmd)
	if err != nil {
		return resp, fmt.Errorf("执行 docker 命令失败: %v", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return resp, nil
	}

	// 先获取 docker stats 信息（只针对运行中的容器）
	// 格式: Name|CPUPerc|MemUsage|MemPerc
	statsCmd := `docker stats --no-stream --format "{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}" 2>/dev/null || echo ""`
	statsOutput, _ := client.ExecuteCommand(statsCmd)
	statsMap := make(map[string]struct {
		CPU      string
		MemUsage string
		MemPerc  string
	})

	if statsOutput = strings.TrimSpace(statsOutput); statsOutput != "" {
		for _, line := range strings.Split(statsOutput, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 4)
			if len(parts) >= 4 {
				statsMap[parts[0]] = struct {
					CPU      string
					MemUsage string
					MemPerc  string
				}{
					CPU:      parts[1],
					MemUsage: parts[2],
					MemPerc:  parts[3],
				}
			}
		}
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 4 {
			continue
		}

		container := model.DockerContainerStatus{
			ContainerId: parts[0],
			Name:        parts[1],
			Image:       parts[2],
			Status:      parts[3],
		}

		if len(parts) > 4 {
			container.Ports = parts[4]
		}
		if len(parts) > 5 {
			container.Created = parts[5]
		}
		if len(parts) > 6 {
			container.RunningFor = parts[6]
		}

		// 添加资源使用信息
		if stats, ok := statsMap[container.Name]; ok {
			container.CPUPercent = stats.CPU
			container.MemUsage = stats.MemUsage
			container.MemPercent = stats.MemPerc
		}

		resp.Containers = append(resp.Containers, container)
	}

	return resp, nil
}

// ToggleServerStatus 切换服务器状态
func ToggleServerStatus(serverId int) error {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("服务器不存在")
	}

	newStatus := 1
	if server.Status == 1 {
		newStatus = 0
	}

	_, err = dbs.DBAdmin.Table("servers").Where("id = ?", serverId).Update(map[string]interface{}{
		"status":     newStatus,
		"updated_at": time.Now(),
	})
	return err
}
