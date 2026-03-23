package control

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 服务名到 systemctl 服务名的映射
var serviceSystemdNames = map[ServiceName]string{
	ServiceServer:   "server.service",
	ServiceWuKongIM: "wukongim.service",
	ServiceGost:     "gost.service",
}

// 服务 Docker 容器名映射
var serviceDockerNames = map[ServiceName]string{
	ServiceServer:   "tsdd-server",
	ServiceWuKongIM: "tsdd-wukongim",
	ServiceGost:     "gost",
}

// 服务配置文件路径映射（Docker 部署：宿主机路径）
var serviceConfigPaths = map[ServiceName]string{
	ServiceServer:   "/opt/tsdd/configs/tsdd.yaml",
	ServiceWuKongIM: "/var/lib/docker/volumes/tsdd_wukongim_data/_data/wk.yaml",
	ServiceGost:     "/etc/gost/config.yaml",
}

// 服务二进制上传路径映射
var serviceUploadPaths = map[ServiceName]string{
	ServiceServer:   "/opt/tsdd/",
	ServiceWuKongIM: "/root/wukongim/",
}

// 服务可执行文件名映射
var serviceBinaryNames = map[ServiceName]string{
	ServiceServer:   "TangSengDaoDaoServer",
	ServiceWuKongIM: "WuKongIM",
}

// 服务默认端口映射（用于服务发现）
var serviceDefaultPorts = map[ServiceName]int{
	ServiceServer:   8090,
	ServiceWuKongIM: 5001,
	ServiceGost:     8080,
}

// ValidateServiceName 验证服务名是否合法
func ValidateServiceName(svc ServiceName) error {
	if _, ok := serviceSystemdNames[svc]; !ok {
		return fmt.Errorf("不支持的服务: %s，仅支持 server/wukongim/gost", svc)
	}
	return nil
}

// buildServiceActionCommand 构建服务操作命令
// 优先尝试 Docker，回退到 systemd
func buildServiceActionCommand(svc ServiceName, action ServiceAction) string {
	dockerName := serviceDockerNames[svc]
	systemdName := serviceSystemdNames[svc]

	// 先检测 Docker 容器是否存在，存在则用 docker，否则回退 systemd
	return fmt.Sprintf(`
docker inspect %s >/dev/null 2>&1
if [ $? -eq 0 ]; then
    docker %s %s
else
    systemctl %s %s
fi
`, dockerName, action, dockerName, action, systemdName)
}

// buildServiceStatusCommand 构建服务状态查询命令
func buildServiceStatusCommand(svc ServiceName) string {
	systemdName := serviceSystemdNames[svc]
	dockerName := serviceDockerNames[svc]

	// 先尝试 systemd，再尝试 docker
	return fmt.Sprintf(`
# 尝试 systemctl
SVC_STATUS=$(systemctl is-active %s 2>/dev/null || echo 'inactive')
if [ "$SVC_STATUS" = "active" ]; then
    PID=$(systemctl show %s --property=MainPID --value 2>/dev/null)
    if [ -n "$PID" ] && [ "$PID" != "0" ]; then
        STATS=$(ps -p $PID -o %%cpu=,%%mem=,rss=,etime= 2>/dev/null | head -1)
    fi
    echo "STATUS=running|PID=$PID|STATS=$STATS"
    exit 0
fi

# 回退 docker
DOCKER_INFO=$(docker inspect --format '{{.State.Status}}|{{.State.Pid}}|{{.State.StartedAt}}' %s 2>/dev/null)
if [ $? -eq 0 ] && echo "$DOCKER_INFO" | grep -q "^running"; then
    DOCKER_STATS=$(docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}' %s 2>/dev/null)
    echo "DOCKER=$DOCKER_INFO|DSTATS=$DOCKER_STATS"
    exit 0
fi

echo "STATUS=stopped"
`, systemdName, systemdName, dockerName, dockerName)
}

// parseServiceStatusOutput 解析服务状态输出
func parseServiceStatusOutput(svc ServiceName, output string) ServiceStatus {
	status := ServiceStatus{
		ServiceName: string(svc),
		Status:      "stopped",
	}
	output = strings.TrimSpace(output)

	if strings.HasPrefix(output, "STATUS=running") {
		status.Status = "running"
		// 解析 PID 和资源
		for _, part := range strings.Split(output, "|") {
			if strings.HasPrefix(part, "PID=") {
				if pid, err := strconv.Atoi(strings.TrimPrefix(part, "PID=")); err == nil {
					status.Pid = pid
				}
			}
			if strings.HasPrefix(part, "STATS=") {
				fields := strings.Fields(strings.TrimPrefix(part, "STATS="))
				if len(fields) >= 4 {
					status.CPU = fields[0] + "%"
					if rss, err := strconv.ParseFloat(fields[2], 64); err == nil {
						status.Memory = formatMemorySize(rss)
					}
					status.Uptime = fields[3]
				}
			}
		}
	} else if strings.HasPrefix(output, "DOCKER=") {
		dockerInfo := strings.TrimPrefix(output, "DOCKER=")
		parts := strings.SplitN(dockerInfo, "|", 5)
		if len(parts) >= 1 && parts[0] == "running" {
			status.Status = "running"
			if len(parts) >= 2 {
				if pid, err := strconv.Atoi(parts[1]); err == nil {
					status.Pid = pid
				}
			}
			if len(parts) >= 3 {
				if startedAt, err := time.Parse(time.RFC3339Nano, parts[2]); err == nil {
					status.Uptime = formatDuration(time.Since(startedAt))
				}
			}
			// 解析 docker stats
			if len(parts) >= 5 && strings.HasPrefix(parts[4], "DSTATS=") {
				dstats := strings.TrimPrefix(parts[4], "DSTATS=")
				dsParts := strings.SplitN(dstats, "|", 2)
				if len(dsParts) >= 2 {
					status.CPU = dsParts[0]
					status.Memory = dsParts[1]
				}
			}
		}
	}

	return status
}

// buildServerStatsCommand 获取服务器资源使用命令
func buildServerStatsCommand() string {
	return `
CPU=$(top -bn1 2>/dev/null | grep "Cpu(s)" | awk '{print $2}' || echo "N/A")
MEM_INFO=$(free -m 2>/dev/null | awk '/^Mem:/{printf "%s|%s", $3, $2}' || echo "N/A|N/A")
DISK_INFO=$(df -h / 2>/dev/null | awk 'NR==2{printf "%s|%s", $3, $2}' || echo "N/A|N/A")
LOAD=$(cat /proc/loadavg 2>/dev/null | awk '{print $1, $2, $3}' || echo "N/A")
echo "CPU=${CPU}|MEM=${MEM_INFO}|DISK=${DISK_INFO}|LOAD=${LOAD}"
`
}

// parseServerStats 解析服务器资源输出
func parseServerStats(output string) ServerStats {
	stats := ServerStats{}
	output = strings.TrimSpace(output)
	for _, part := range strings.Split(output, "|") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "CPU":
			stats.CPUUsage = kv[1] + "%"
		case "MEM":
			memParts := strings.SplitN(kv[1], "|", 2)
			if len(memParts) == 2 {
				stats.MemoryUsage = memParts[0] + "MB"
				stats.MemoryTotal = memParts[1] + "MB"
			}
		case "DISK":
			diskParts := strings.SplitN(kv[1], "|", 2)
			if len(diskParts) == 2 {
				stats.DiskUsage = diskParts[0]
				stats.DiskTotal = diskParts[1]
			}
		case "LOAD":
			stats.LoadAvg = kv[1]
		}
	}
	return stats
}

// buildDockerContainersCommand Docker 容器列表命令
func buildDockerContainersCommand() string {
	return `docker ps -a --format "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}|{{.CreatedAt}}|{{.RunningFor}}" 2>/dev/null || echo ""`
}

// buildDockerStatsCommand Docker 容器资源使用命令
func buildDockerStatsCommand() string {
	return `docker stats --no-stream --format "{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}" 2>/dev/null || echo ""`
}

// parseDockerContainers 解析 Docker 容器列表输出
func parseDockerContainers(listOutput, statsOutput string) []DockerContainer {
	listOutput = strings.TrimSpace(listOutput)
	if listOutput == "" {
		return nil
	}

	// 先构建 stats map
	statsMap := make(map[string][3]string) // name -> [cpu, memUsage, memPerc]
	statsOutput = strings.TrimSpace(statsOutput)
	if statsOutput != "" {
		for _, line := range strings.Split(statsOutput, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "|", 4)
			if len(parts) >= 4 {
				statsMap[parts[0]] = [3]string{parts[1], parts[2], parts[3]}
			}
		}
	}

	var containers []DockerContainer
	for _, line := range strings.Split(listOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 4 {
			continue
		}
		c := DockerContainer{
			ContainerId: parts[0],
			Name:        parts[1],
			Image:       parts[2],
			Status:      parts[3],
		}
		if len(parts) > 4 {
			c.Ports = parts[4]
		}
		if len(parts) > 5 {
			c.Created = parts[5]
		}
		if len(parts) > 6 {
			c.RunningFor = parts[6]
		}
		if s, ok := statsMap[c.Name]; ok {
			c.CPUPercent = s[0]
			c.MemUsage = s[1]
			c.MemPercent = s[2]
		}
		containers = append(containers, c)
	}
	return containers
}

// --- 辅助函数 ---

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

func formatMemorySize(kb float64) string {
	if kb < 1024 {
		return fmt.Sprintf("%.0fKB", kb)
	} else if kb < 1024*1024 {
		return fmt.Sprintf("%.1fMB", kb/1024)
	}
	return fmt.Sprintf("%.2fGB", kb/1024/1024)
}
