package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/internal/server/utils"
	"strings"
	"sync"
)

// formatMemory 格式化内存显示（KB转为MB/GB）
func formatMemory(kb string) string {
	var kbVal float64
	fmt.Sscanf(kb, "%f", &kbVal)

	if kbVal < 1024 {
		return fmt.Sprintf("%.0fKB", kbVal)
	} else if kbVal < 1024*1024 {
		return fmt.Sprintf("%.1fMB", kbVal/1024)
	} else {
		return fmt.Sprintf("%.2fGB", kbVal/1024/1024)
	}
}

// parseFloat 解析浮点数字符串
func parseFloat(s string) float64 {
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

// getServerStats 获取服务器整体资源使用情况（内部方法）
func getServerStats(client *utils.SSHClient) *model.ServerStatsResp {
	stats := &model.ServerStatsResp{}

	// 检测操作系统类型
	osType := detectOS(client)

	// 1. CPU使用率（使用 vmstat，输出格式稳定）
	if osType == "darwin" {
		// macOS: 使用 iostat（vmstat 格式不同）
		cpuCmd := "iostat -c 2 -w 1 2>/dev/null | tail -1 | awk '{print 100-$6}'"
		cpuOutput := sanitizeOutput(client.ExecuteCommandSilent(cpuCmd))
		if cpuOutput != "" && !strings.Contains(cpuOutput, "command not found") {
			cpuVal := parseFloat(strings.TrimSpace(cpuOutput))
			stats.CPUUsage = fmt.Sprintf("%.1f%%", cpuVal)
		}
	} else {
		// Linux: 优先使用 vmstat（最稳定）
		// vmstat 1 2: 采样2次，每次1秒，取第二次（更准确）
		cpuCmd := "LC_ALL=C LANG=C vmstat 1 2 2>/dev/null | tail -1 | awk '{print 100-$15}'"
		cpuOutput := sanitizeOutput(client.ExecuteCommandSilent(cpuCmd))

		if cpuOutput != "" && !strings.Contains(cpuOutput, "command not found") {
			cpuVal := parseFloat(strings.TrimSpace(cpuOutput))
			stats.CPUUsage = fmt.Sprintf("%.1f%%", cpuVal)
		}
	}

	// 2. 内存使用情况
	if osType == "darwin" {
		// macOS: 使用 vm_stat
		memCmd := "memory_pressure 2>/dev/null | grep 'System-wide memory free percentage' | awk '{print $5}' | sed 's/%//'"
		memOutput := client.ExecuteCommandSilent(memCmd)
		if memOutput == "" {
			// 备用方案：使用 top
			memCmd = "top -l 1 | grep PhysMem | awk '{print $2, $6}'"
			memOutput = client.ExecuteCommandSilent(memCmd)
			if memOutput != "" {
				parts := strings.Fields(strings.TrimSpace(memOutput))
				if len(parts) >= 2 {
					stats.MemoryUsage = parts[0]
					stats.MemoryTotal = parts[1]
				}
			}
		}
	} else {
		// Linux: 使用 free
		memCmd := "LC_ALL=C LANG=C free -h 2>/dev/null | grep Mem | awk '{print $3, $2}'"
		memOutput := sanitizeOutput(client.ExecuteCommandSilent(memCmd))
		if memOutput != "" {
			parts := strings.Fields(strings.TrimSpace(memOutput))
			if len(parts) >= 2 {
				stats.MemoryUsage = parts[0]
				stats.MemoryTotal = parts[1]
			}
		}
	}

	// 3. 磁盘使用情况（根目录）
	diskCmd := "LC_ALL=C LANG=C df -h / 2>/dev/null | tail -1 | awk '{print $3, $2, $5}'"
	diskOutput := sanitizeOutput(client.ExecuteCommandSilent(diskCmd))
	if diskOutput != "" {
		parts := strings.Fields(strings.TrimSpace(diskOutput))
		if len(parts) >= 3 {
			stats.DiskUsage = parts[0] + " / " + parts[1]
			stats.DiskTotal = parts[2]
		}
	}

	// 4. 系统负载
	loadCmd := "LC_ALL=C LANG=C uptime 2>/dev/null"
	loadOutput := sanitizeOutput(client.ExecuteCommandSilent(loadCmd))
	if loadOutput != "" {
		// 提取 load average 部分
		if idx := strings.Index(loadOutput, "load average"); idx >= 0 {
			loadPart := loadOutput[idx+len("load average"):]
			loadPart = strings.TrimLeft(loadPart, ": ")
			// 提取前三个数字
			parts := strings.Split(loadPart, ",")
			if len(parts) >= 3 {
				load1 := strings.TrimSpace(parts[0])
				load5 := strings.TrimSpace(parts[1])
				load15 := strings.TrimSpace(parts[2])
				stats.LoadAvg = fmt.Sprintf("%s, %s, %s", load1, load5, load15)
			}
		}
	}

	return stats
}

// sanitizeOutput 过滤掉可能干扰解析的本地 bash locale 警告等文本
func sanitizeOutput(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		l := strings.ToLower(t)
		// 过滤常见的噪音与错误输出：
		// - bash 前缀错误
		// - locale 警告
		// - STDERR 汇总标记
		// - 非交互 shell 误输出（.bashrc/.profile 等）
		// - 常见的 not found/permission denied/no such file 等
		if strings.HasPrefix(l, "bash:") ||
			strings.HasPrefix(l, "stderr:") ||
			strings.Contains(l, "setlocale") ||
			strings.Contains(l, "warning:") ||
			strings.Contains(l, ".bashrc") ||
			strings.Contains(l, ".bash_profile") ||
			strings.Contains(l, ".profile") ||
			strings.Contains(l, "command not found") ||
			strings.Contains(l, "not found") ||
			strings.Contains(l, "no such file") ||
			strings.Contains(l, "permission denied") {
			continue
		}
		kept = append(kept, t)
	}
	return strings.Join(kept, "\n")
}

// detectOS 检测操作系统类型
func detectOS(client *utils.SSHClient) string {
	// 检测是否为 macOS
	output := client.ExecuteCommandSilent("uname -s")
	osName := strings.ToLower(strings.TrimSpace(output))

	if strings.Contains(osName, "darwin") {
		return "darwin"
	}
	return "linux"
}

// GetServerStats 获取服务器资源使用情况（对外接口）
func GetServerStats(req model.GetServerStatsReq) (model.ServerStatsResp, error) {
	var resp model.ServerStatsResp

	// 获取SSH客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	stats := getServerStats(client.SSHClient)
	if stats != nil {
		resp = *stats
	}

	return resp, nil
}

// GetServerStatsBatch 并发获取多个服务器的基础资源（CPU、内存）
func GetServerStatsBatch(req model.GetServerStatsBatchReq) (model.GetServerStatsBatchResp, error) {
	var resp model.GetServerStatsBatchResp
	if len(req.ServerIds) == 0 {
		return resp, nil
	}

	// 限制并发数，避免瞬时过高并发
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sid := range req.ServerIds {
		serverID := sid
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			stat := model.ServerBasicStat{ServerId: serverID}

			// 获取SSH客户端
			client, err := GetSSHClient(serverID)
			if err != nil {
				stat.Error = err.Error()
			} else {
				s := getServerStats(client.SSHClient)
				if s != nil {
					stat.CPUUsage = s.CPUUsage
					stat.MemoryUsage = s.MemoryUsage
					stat.MemoryTotal = s.MemoryTotal
				} else {
					stat.Error = "failed to get stats"
				}
			}

			mu.Lock()
			resp.Stats = append(resp.Stats, stat)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return resp, nil
}
