package deploy

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// serviceDockerNames 使用 model 中的统一映射
var serviceDockerNames = model.ServiceDockerNames

// 服务配置文件路径映射（Docker 部署：宿主机上的路径）
// 按优先级排列，GetConfigFile 会依次尝试
var serviceConfigPaths = map[string][]string{
	"server":   {"/opt/tsdd/configs/tsdd.yaml"},
	"wukongim": {"/data/db/wukongim/wk.yaml"},
	"gost":     {"/etc/gost/config.yaml"},
}

// ServiceAction 执行服务操作（start/stop/restart）
// 优先使用 Docker，回退到 systemd
func ServiceAction(req model.ServiceActionReq, operator string) (model.ServiceActionResp, error) {
	var resp model.ServiceActionResp

	// 验证服务名
	if _, ok := serviceSystemdNames[req.ServiceName]; !ok {
		return resp, fmt.Errorf("不支持的服务: %s，仅支持 server/wukongim/gost", req.ServiceName)
	}

	// 获取 SSH 客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 优先尝试 Docker
	var output string
	var execErr error
	dockerName := serviceDockerNames[req.ServiceName]
	if dockerName != "" {
		// 检查 Docker 容器是否存在
		checkCmd := fmt.Sprintf("docker inspect %s >/dev/null 2>&1 && echo 'exists'", dockerName)
		checkOutput, _ := client.ExecuteCommand(checkCmd)
		if strings.TrimSpace(checkOutput) == "exists" {
			cmd := fmt.Sprintf("docker %s %s", req.Action, dockerName)
			output, execErr = client.ExecuteCommand(cmd)
		} else {
			// Docker 容器不存在，回退到 systemd
			systemdName := serviceSystemdNames[req.ServiceName]
			cmd := fmt.Sprintf("systemctl %s %s", req.Action, systemdName)
			output, execErr = client.ExecuteCommand(cmd)
		}
	} else {
		// 无 Docker 映射，直接用 systemd
		systemdName := serviceSystemdNames[req.ServiceName]
		cmd := fmt.Sprintf("systemctl %s %s", req.Action, systemdName)
		output, execErr = client.ExecuteCommand(cmd)
	}

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
		status := queryServiceStatusWithName(client.SSHClient, systemdName, svc)
		status.ServiceName = svc
		resp.Services = append(resp.Services, status)
	}

	return resp, nil
}

// queryServiceStatus 查询单个服务的状态（支持 systemd 和 Docker 两种部署方式）
func queryServiceStatus(client *utils.SSHClient, systemdName string) model.ServiceStatusResp {
	return queryServiceStatusWithName(client, systemdName, "")
}

// queryServiceStatusWithName 查询单个服务的状态，serviceName 用于 Docker 回退检查
func queryServiceStatusWithName(client *utils.SSHClient, systemdName string, serviceName string) model.ServiceStatusResp {
	var status model.ServiceStatusResp

	// 1. 先尝试 systemctl 检查
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
		return status
	}

	// 2. systemd 未运行，回退检查 Docker 容器
	dockerName := serviceDockerNames[serviceName]
	if dockerName != "" {
		dockerCmd := fmt.Sprintf(`docker inspect --format '{{.State.Status}}|{{.State.Pid}}|{{.State.StartedAt}}' %s 2>/dev/null`, dockerName)
		dockerOutput, dockerErr := client.ExecuteCommand(dockerCmd)
		dockerOutput = strings.TrimSpace(dockerOutput)
		if dockerErr == nil && dockerOutput != "" {
			parts := strings.SplitN(dockerOutput, "|", 3)
			if len(parts) >= 1 && parts[0] == "running" {
				status.Status = "running"
				if len(parts) >= 2 {
					if pid, err := strconv.Atoi(parts[1]); err == nil && pid > 0 {
						status.Pid = pid
						// 获取容器资源使用
						statsCmd := fmt.Sprintf(`docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}' %s 2>/dev/null`, dockerName)
						statsOutput, _ := client.ExecuteCommand(statsCmd)
						statsOutput = strings.TrimSpace(statsOutput)
						if statsOutput != "" {
							statsParts := strings.SplitN(statsOutput, "|", 2)
							if len(statsParts) >= 2 {
								status.CPU = statsParts[0]
								status.Memory = statsParts[1]
							}
						}
					}
				}
				if len(parts) >= 3 {
					// 计算运行时间
					startedAt, err := time.Parse(time.RFC3339Nano, parts[2])
					if err == nil {
						status.Uptime = formatDuration(time.Since(startedAt))
					}
				}
				return status
			}
		}
	}

	status.Status = "stopped"
	return status
}

// formatDuration 格式化持续时间为可读字符串
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
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

	configPaths, ok := serviceConfigPaths[serviceName]
	if !ok {
		return resp, fmt.Errorf("服务 %s 没有配置文件", serviceName)
	}

	client, err := GetSSHClient(serverId)
	if err != nil {
		return resp, err
	}

	// 依次尝试候选路径，找到第一个存在的
	var configPath string
	for _, p := range configPaths {
		checkCmd := fmt.Sprintf("sudo test -f '%s' && echo 'exists'", p)
		checkOutput, _ := client.ExecuteCommand(checkCmd)
		if strings.TrimSpace(checkOutput) == "exists" {
			configPath = p
			break
		}
	}

	if configPath == "" {
		// 所有路径都不存在，返回第一个路径（允许用户创建）
		resp.ServiceName = serviceName
		resp.ConfigPath = configPaths[0]
		resp.Content = ""
		return resp, nil
	}

	cmd := fmt.Sprintf("sudo cat '%s'", configPath)
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

	configPaths, ok := serviceConfigPaths[serviceName]
	if !ok {
		return resp, fmt.Errorf("服务 %s 没有配置文件", serviceName)
	}

	client, err := GetSSHClient(serverId)
	if err != nil {
		return resp, err
	}

	// 查找已存在的配置文件路径，不存在则用第一个候选路径
	configPath := configPaths[0]
	for _, p := range configPaths {
		checkCmd := fmt.Sprintf("sudo test -f '%s' && echo 'exists'", p)
		checkOutput, _ := client.ExecuteCommand(checkCmd)
		if strings.TrimSpace(checkOutput) == "exists" {
			configPath = p
			break
		}
	}

	// 备份原配置
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", configPath, ts)
	client.ExecuteCommand(fmt.Sprintf("sudo cp -f '%s' '%s' 2>/dev/null", configPath, backupPath))

	// 确保目录存在
	dir := path.Dir(configPath)
	client.ExecuteCommand(fmt.Sprintf("sudo mkdir -p '%s'", dir))

	// 写入新配置
	escapedContent := strings.ReplaceAll(content, "'", "'\"'\"'")
	cmd := fmt.Sprintf("echo '%s' | sudo tee '%s' > /dev/null", escapedContent, configPath)
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

	// 如果是 server 服务，自动更新 AMI 部署资源包
	if serviceName == "server" {
		go updateTSDDResourcesTarball(localPath)
	}

	return localPath, nil
}

// updateTSDDResourcesTarball 异步更新 /opt/control/tsdd-resources.tar.gz
// 将最新的 TangSengDaoDaoServer 二进制和 assets 打包，供 AMI 新部署使用
func updateTSDDResourcesTarball(newBinaryPath string) {
	const resourcesDir = "/opt/control/tsdd-resources-tmp"
	const tarballPath = "/opt/control/tsdd-resources.tar.gz"
	const assetsDir = "/opt/control/tsdd-resources-assets"

	// 创建临时打包目录
	os.RemoveAll(resourcesDir)
	os.MkdirAll(resourcesDir+"/assets/assets", 0755)

	// 复制新二进制
	src, err := os.Open(newBinaryPath)
	if err != nil {
		return
	}
	dst, err := os.Create(resourcesDir + "/TangSengDaoDaoServer")
	if err != nil {
		src.Close()
		return
	}
	io.Copy(dst, src)
	src.Close()
	dst.Close()
	os.Chmod(resourcesDir+"/TangSengDaoDaoServer", 0755)

	// 复制 assets（从已有的资源包解压或从固定目录）
	if entries, err := os.ReadDir(assetsDir + "/assets"); err == nil {
		for _, e := range entries {
			s, _ := os.Open(assetsDir + "/assets/" + e.Name())
			if s != nil {
				d, _ := os.Create(resourcesDir + "/assets/assets/" + e.Name())
				if d != nil {
					io.Copy(d, s)
					d.Close()
				}
				s.Close()
			}
		}
	} else {
		// 从现有 tarball 中提取 assets
		os.MkdirAll(resourcesDir+"/extract", 0755)
		extractCmd := fmt.Sprintf("tar xzf %s -C %s/extract assets/ 2>/dev/null && cp -r %s/extract/assets/* %s/assets/ 2>/dev/null",
			tarballPath, resourcesDir, resourcesDir, resourcesDir)
		exec.Command("bash", "-c", extractCmd).Run()
	}

	// 打包
	tarCmd := fmt.Sprintf("cd %s && tar czf %s TangSengDaoDaoServer assets/", resourcesDir, tarballPath)
	exec.Command("bash", "-c", tarCmd).Run()

	// 清理
	os.RemoveAll(resourcesDir)
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
			dockerName := model.ServiceDockerNames[req.ServiceName]

			// 确保 docker-compose 有二进制的 volume 挂载，然后重启
			restartScript := fmt.Sprintf(`
cd %s

# 确保有二进制 volume 挂载（server 服务）
NEEDS_RECREATE=0
if [ "%s" = "server" ] && [ -f docker-compose.yml ]; then
  if ! grep -q '/home/app' docker-compose.yml; then
    # 添加 volume 挂载（正确的容器内路径）
    if grep -q '/home/assets' docker-compose.yml; then
      # 已有 assets 挂载，在其后添加二进制挂载
      sed -i '/\/home\/assets/a\      - ./%s:/home/app' docker-compose.yml
    elif grep -q 'condition: service_started' docker-compose.yml; then
      # 没有任何自定义挂载，添加完整 volumes 段
      sed -i '/condition: service_started/a\    volumes:\n      - ./configs/hxd.yaml:/home/configs/hxd.yaml:ro\n      - ./assets:/home/assets:ro\n      - ./%s:/home/app' docker-compose.yml
    fi
    NEEDS_RECREATE=1
  fi
fi

# 重启容器
if [ "$NEEDS_RECREATE" = "1" ]; then
  docker compose up -d %s 2>/dev/null || docker-compose up -d %s 2>/dev/null
else
  docker restart %s 2>/dev/null
fi
echo "restart done"
`, remoteUploadPath,
				req.ServiceName,
				binaryName,
				binaryName,
				dockerName, dockerName,
				dockerName)

			_, restartErr := targetClient.ExecuteCommand(restartScript)
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

// BatchSyncConfig 批量同步 docker-compose 配置到已部署的服务器
// 从 cluster_nodes 表读取集群参数，用 Go 模板重新生成 compose + .env，推送到服务器并 docker compose up -d
func BatchSyncConfig(req model.BatchSyncConfigReq, operator string) (model.BatchSyncConfigResp, error) {
	var resp model.BatchSyncConfigResp
	resp.Results = make([]model.SyncConfigResult, 0, len(req.ServerIds))
	resp.TotalCount = len(req.ServerIds)

	for _, serverId := range req.ServerIds {
		result := model.SyncConfigResult{ServerId: serverId}

		// 查服务器信息
		var server entity.Servers
		has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
		if err != nil || !has {
			result.Message = "服务器不存在"
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}
		result.ServerName = server.Name
		result.ServerHost = server.Host

		// 查集群节点信息
		var node entity.ClusterNodes
		has, err = dbs.DBAdmin.Where("server_id = ?", serverId).Get(&node)
		if err != nil || !has {
			result.Message = "未找到集群节点信息，请先通过 Control 面板部署此服务器"
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}
		result.NodeRole = node.NodeRole

		// 获取 SSH 客户端
		sshWrapper, err := GetSSHClient(serverId)
		if err != nil {
			result.Message = fmt.Sprintf("SSH 连接失败: %v", err)
			resp.Results = append(resp.Results, result)
			resp.FailCount++
			continue
		}

		// 构建 DeployConfig
		config := model.DefaultDeployConfig
		config.ExternalIP = server.Host
		config.NodeRole = node.NodeRole
		config.DBHost = node.DBHost
		config.RedisHost = node.DBHost
		config.MinioHost = node.MinioHost
		config.MinioPublicHost = lookupMinioPublicIP(node.MerchantId)
		config.WKNodeId = node.WKNodeId
		config.ControlAPIUsername = "merchant_api"
		config.ControlAPIPassword = "MerchantAPI@2026"

		// 检测内网 IP（优先实时检测，回退到数据库记录）
		if privateIP := DetectPrivateIP(sshWrapper.SSHClient); privateIP != "" {
			config.PrivateIP = privateIP
		} else if node.PrivateIP != "" {
			config.PrivateIP = node.PrivateIP
		} else {
			config.PrivateIP = server.Host
		}

		// 查找种子节点（同商户的其他 app 节点中 wk_node_id 最小的）
		if node.NodeRole == "app" && node.WKNodeId > 0 {
			var seedNode entity.ClusterNodes
			hasSeed, _ := dbs.DBAdmin.Where("merchant_id = ? AND node_role = 'app' AND wk_node_id < ? AND server_id != ?",
				node.MerchantId, node.WKNodeId, serverId).OrderBy("wk_node_id ASC").Limit(1).Get(&seedNode)
			if hasSeed && seedNode.PrivateIP != "" {
				config.WKSeedNode = fmt.Sprintf("%d@%s:11110", seedNode.WKNodeId, seedNode.PrivateIP)
			}
		}

		// 生成配置
		composeContent := GenerateComposeByRole(config)
		envContent := GenerateEnvByRole(config)

		// SSH 推送：备份 → 写入 → docker compose up -d
		syncScript := fmt.Sprintf(`cd /opt/tsdd
# 备份原配置
ts=$(date +%%Y%%m%%d_%%H%%M%%S)
[ -f docker-compose.yml ] && cp docker-compose.yml docker-compose.yml.${ts}.bak
[ -f .env ] && cp .env .env.${ts}.bak

# 写入新配置
cat > docker-compose.yml << 'COMPOSE_EOF'
%s
COMPOSE_EOF

cat > .env << 'ENV_EOF'
%s
ENV_EOF

# 应用变更（Docker 自动只重建有变化的容器）
docker compose up -d 2>&1 || docker-compose up -d 2>&1
sleep 3
echo "=== 容器状态 ==="
docker ps --format "{{.Names}}\t{{.Status}}" | grep tsdd || echo "无 tsdd 容器"
`, composeContent, envContent)

		output, err := sshWrapper.ExecuteCommandWithTimeout(syncScript, 2*time.Minute)
		if err != nil {
			result.Message = fmt.Sprintf("推送配置失败: %v", err)
			resp.Results = append(resp.Results, result)
			resp.FailCount++
		} else {
			result.Success = true
			// 提取容器状态摘要（只统计 "=== 容器状态 ===" 之后的 docker ps 输出）
			lines := strings.Split(strings.TrimSpace(output), "\n")
			containerCount := 0
			inStatus := false
			for _, l := range lines {
				if strings.Contains(l, "=== 容器状态 ===") {
					inStatus = true
					continue
				}
				if inStatus && strings.Contains(l, "tsdd-") {
					containerCount++
				}
			}
			result.Message = fmt.Sprintf("配置已同步，%d 个容器运行中", containerCount)
			resp.Results = append(resp.Results, result)
			resp.SuccessCount++
		}

		// 记录操作历史
		history := entity.DeployHistory{
			ServerId:    serverId,
			Action:      "sync_config",
			ServiceName: "docker-compose",
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
