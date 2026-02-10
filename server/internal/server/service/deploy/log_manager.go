package deploy

import (
	"fmt"
	"server/internal/server/model"
	"strings"
	"time"
)

// QueryLogs 统一日志查询
func QueryLogs(req model.LogQueryReq) (model.LogQueryResp, error) {
	var resp model.LogQueryResp

	// 获取 SSH 客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, fmt.Errorf("获取SSH连接失败: %v", err)
	}

	// 设置默认值
	if req.Lines <= 0 {
		req.Lines = 100
	}
	if req.Lines > 5000 {
		req.Lines = 5000
		resp.Truncated = true
	}
	if req.QueryType == "" {
		req.QueryType = "journalctl"
	}

	// 构建命令
	var cmd string
	switch req.QueryType {
	case "journalctl":
		cmd = buildJournalctlCmd(req)
	case "docker":
		cmd = buildDockerLogsCmd(req)
	case "file":
		cmd = buildFileLogsCmd(req)
	default:
		cmd = buildJournalctlCmd(req)
	}

	resp.Command = cmd

	// 执行命令
	output, err := client.ExecuteCommandWithTimeout(cmd, 30*time.Second)
	if err != nil {
		// 即使有错误也返回部分输出
		resp.Logs = output
		resp.LineCount = len(strings.Split(output, "\n"))
		return resp, fmt.Errorf("执行命令失败: %v", err)
	}

	resp.Logs = output
	resp.LineCount = len(strings.Split(strings.TrimSpace(output), "\n"))
	if output == "" {
		resp.LineCount = 0
	}

	return resp, nil
}

// buildJournalctlCmd 构建 journalctl 命令
func buildJournalctlCmd(req model.LogQueryReq) string {
	parts := []string{"journalctl"}

	// 服务名
	if req.ServiceName != "" {
		systemdName, ok := serviceSystemdNames[req.ServiceName]
		if ok {
			parts = append(parts, "-u", systemdName)
		} else {
			parts = append(parts, "-u", req.ServiceName+".service")
		}
	}

	// 行数
	if req.Lines > 0 {
		parts = append(parts, "-n", fmt.Sprintf("%d", req.Lines))
	}

	// 时间范围
	if req.Since != "" {
		// 支持 "1h", "30m" 格式，转换为 journalctl 格式
		since := convertTimeFormat(req.Since)
		parts = append(parts, "--since", since)
	}
	if req.Until != "" {
		until := convertTimeFormat(req.Until)
		parts = append(parts, "--until", until)
	}

	parts = append(parts, "--no-pager")

	cmd := strings.Join(parts, " ")

	// 添加过滤
	cmd = addFilters(cmd, req.Keyword, req.Level)

	return cmd
}

// buildDockerLogsCmd 构建 docker logs 命令
func buildDockerLogsCmd(req model.LogQueryReq) string {
	containerName := req.ContainerName
	if containerName == "" {
		containerName = req.ServiceName
	}
	if containerName == "" {
		return "echo 'container_name is required for docker logs'"
	}

	parts := []string{"docker", "logs"}

	if req.Lines > 0 {
		parts = append(parts, "--tail", fmt.Sprintf("%d", req.Lines))
	}

	if req.Since != "" {
		parts = append(parts, "--since", req.Since)
	}

	parts = append(parts, containerName, "2>&1")

	cmd := strings.Join(parts, " ")

	// 添加过滤
	cmd = addFilters(cmd, req.Keyword, req.Level)

	return cmd
}

// buildFileLogsCmd 构建文件日志查询命令
func buildFileLogsCmd(req model.LogQueryReq) string {
	logPath := req.LogPath
	if logPath == "" {
		// 根据服务名使用默认路径
		switch req.ServiceName {
		case "server":
			logPath = "/root/server/logs/*.log"
		case "wukongim":
			logPath = "/root/wukongim/logs/*.log"
		case "gost":
			logPath = "/root/gost/logs/*.log"
		default:
			return "echo 'log_path is required for file logs'"
		}
	}

	parts := []string{"tail", "-n", fmt.Sprintf("%d", req.Lines), logPath}

	cmd := strings.Join(parts, " ")

	// 添加过滤
	cmd = addFilters(cmd, req.Keyword, req.Level)

	return cmd
}

// convertTimeFormat 转换时间格式
// 支持 "1h", "30m", "1d" 格式转换为 journalctl 格式
func convertTimeFormat(input string) string {
	input = strings.TrimSpace(input)

	// 如果已经是日期时间格式，直接返回
	if strings.Contains(input, "-") && strings.Contains(input, ":") {
		return fmt.Sprintf(`"%s"`, input)
	}

	// 处理相对时间格式
	if strings.HasSuffix(input, "h") || strings.HasSuffix(input, "m") || strings.HasSuffix(input, "d") {
		return fmt.Sprintf(`"%s ago"`, input)
	}

	return fmt.Sprintf(`"%s"`, input)
}

// addFilters 添加关键字和级别过滤
func addFilters(cmd, keyword, level string) string {
	if keyword != "" {
		// 转义特殊字符
		safeKeyword := strings.ReplaceAll(keyword, "'", "'\\''")
		cmd += fmt.Sprintf(" | grep -i '%s'", safeKeyword)
	}

	if level != "" {
		// 常见日志级别关键字
		levelPatterns := map[string]string{
			"error":   "error\\|ERROR\\|Error\\|fatal\\|FATAL\\|Fatal",
			"warn":    "warn\\|WARN\\|Warn\\|warning\\|WARNING",
			"info":    "info\\|INFO\\|Info",
			"debug":   "debug\\|DEBUG\\|Debug",
		}
		if pattern, ok := levelPatterns[strings.ToLower(level)]; ok {
			cmd += fmt.Sprintf(" | grep -E '%s'", pattern)
		} else {
			cmd += fmt.Sprintf(" | grep -i '%s'", level)
		}
	}

	return cmd
}
