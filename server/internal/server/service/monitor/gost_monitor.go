package monitor

import (
	"encoding/json"
	"fmt"
	"server/internal/server/service/alert"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	maxCheckConcurrent = 5
	errorThreshold     = 0.05 // 5% 错误率触发告警
)

// GostCheckResult 单台服务器检查结果
type GostCheckResult struct {
	ServerId      int     `json:"server_id"`
	ServerName    string  `json:"server_name"`
	ServerHost    string  `json:"server_host"`
	Status        string  `json:"status"` // up/down/degraded
	ApiReachable  int     `json:"api_reachable"`
	ExpectedPorts int     `json:"expected_ports"`
	ActualPorts   int     `json:"actual_ports"`
	MissingPorts  []int   `json:"missing_ports,omitempty"`
	TotalConns    int64   `json:"total_conns"`
	CurrentConns  int64   `json:"current_conns"`
	InputBytes    int64   `json:"input_bytes"`
	OutputBytes   int64   `json:"output_bytes"`
	TotalErrors   int64   `json:"total_errors"`
	ErrorRate     float64 `json:"error_rate"`
	ErrorMessage  string  `json:"error_message,omitempty"`
	CheckDuration int     `json:"check_duration"` // ms
}

// CheckAllGostServers 检查所有 GOST 系统服务器
func CheckAllGostServers() ([]GostCheckResult, error) {
	// 查所有启用的系统服务器
	var servers []entity.Servers
	err := dbs.DBAdmin.Where("server_type = 2 AND status = 1").Find(&servers)
	if err != nil {
		return nil, fmt.Errorf("查询系统服务器失败: %v", err)
	}
	if len(servers) == 0 {
		return []GostCheckResult{}, nil
	}

	results := make([]GostCheckResult, len(servers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxCheckConcurrent)

	for i, srv := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, server entity.Servers) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = checkServer(server)
		}(i, srv)
	}
	wg.Wait()

	// 写入 DB + 触发告警
	for _, r := range results {
		saveMonitorLog(r)
		triggerAlerts(r)
	}

	return results, nil
}

// checkServer 检查单台 GOST 服务器
func checkServer(server entity.Servers) GostCheckResult {
	start := time.Now()
	r := GostCheckResult{
		ServerId:   server.Id,
		ServerName: server.Name,
		ServerHost: server.Host,
		Status:     "unknown",
	}

	client := gostapi.NewClient()

	// 1. API 连通性
	config, err := client.GetConfig(server.Host, "")
	if err != nil {
		r.Status = "down"
		r.ApiReachable = 0
		r.ErrorMessage = fmt.Sprintf("API 不可达: %s", err.Error())
		r.CheckDuration = int(time.Since(start).Milliseconds())
		return r
	}
	r.ApiReachable = 1

	// 2. 转发规则验证
	checkForwardingRules(server, config, &r)

	// 3. 流量统计
	checkServiceStats(server, config, client, &r)

	// 判断总体状态
	if r.ActualPorts >= r.ExpectedPorts && r.ErrorRate < errorThreshold {
		r.Status = "up"
	} else {
		r.Status = "degraded"
	}

	r.CheckDuration = int(time.Since(start).Milliseconds())
	return r
}

// checkForwardingRules 验证转发规则完整性
func checkForwardingRules(server entity.Servers, config *gostapi.Config, r *GostCheckResult) {
	// 查该服务器关联的商户
	var gostServers []entity.MerchantGostServers
	_ = dbs.DBAdmin.Where("server_id = ? AND status = 1", server.Id).Find(&gostServers)

	if len(gostServers) == 0 {
		return
	}

	merchantIds := make([]int, 0, len(gostServers))
	for _, gs := range gostServers {
		merchantIds = append(merchantIds, gs.MerchantId)
	}

	var merchants []entity.Merchants
	_ = dbs.DBAdmin.In("id", merchantIds).Find(&merchants)

	// 构建期望端口集合：每个商户 3 个端口 (TCP/WS/HTTP)
	expectedPorts := make(map[int]bool)
	for _, m := range merchants {
		if m.Port > 0 {
			expectedPorts[m.Port+gostapi.PortOffsetTCP] = true
			expectedPorts[m.Port+gostapi.PortOffsetWS] = true
			expectedPorts[m.Port+gostapi.PortOffsetHTTP] = true
		}
	}
	r.ExpectedPorts = len(expectedPorts)

	// 从 config 中提取实际转发端口
	actualForwards, err := gostapi.NewClient().GetForwardStatus(server.Host)
	if err != nil {
		logx.Errorf("[GostMonitor] GetForwardStatus(%s) 失败: %v", server.Host, err)
		return
	}
	r.ActualPorts = len(actualForwards)

	// 找出缺失端口
	var missing []int
	for port := range expectedPorts {
		if _, ok := actualForwards[port]; !ok {
			missing = append(missing, port)
		}
	}
	r.MissingPorts = missing
}

// checkServiceStats 汇总所有 GOST 服务的流量统计
func checkServiceStats(server entity.Servers, config *gostapi.Config, client *gostapi.Client, r *GostCheckResult) {
	var totalConns, currentConns, inputBytes, outputBytes, totalErrors uint64

	for _, svc := range config.Services {
		detail, err := client.GetService(server.Host, svc.Name)
		if err != nil {
			continue
		}
		if detail.Status != nil && detail.Status.Stats != nil {
			stats := detail.Status.Stats
			totalConns += stats.TotalConns
			currentConns += stats.CurrentConns
			inputBytes += stats.InputBytes
			outputBytes += stats.OutputBytes
			totalErrors += stats.TotalErrs
		}
	}

	r.TotalConns = int64(totalConns)
	r.CurrentConns = int64(currentConns)
	r.InputBytes = int64(inputBytes)
	r.OutputBytes = int64(outputBytes)
	r.TotalErrors = int64(totalErrors)

	if totalConns > 0 {
		r.ErrorRate = float64(totalErrors) / float64(totalConns)
	}
}

// saveMonitorLog 保存检查结果到数据库
func saveMonitorLog(r GostCheckResult) {
	missingJSON := ""
	if len(r.MissingPorts) > 0 {
		b, _ := json.Marshal(r.MissingPorts)
		missingJSON = string(b)
	}

	log := &entity.GostMonitorLogs{
		ServerId:      r.ServerId,
		ServerName:    r.ServerName,
		ServerHost:    r.ServerHost,
		Status:        r.Status,
		ApiReachable:  r.ApiReachable,
		ExpectedPorts: r.ExpectedPorts,
		ActualPorts:   r.ActualPorts,
		MissingPorts:  missingJSON,
		TotalConns:    r.TotalConns,
		CurrentConns:  r.CurrentConns,
		InputBytes:    r.InputBytes,
		OutputBytes:   r.OutputBytes,
		TotalErrors:   r.TotalErrors,
		ErrorRate:     r.ErrorRate,
		ErrorMessage:  r.ErrorMessage,
		CheckDuration: r.CheckDuration,
		CreatedAt:     time.Now(),
	}

	if _, err := dbs.DBAdmin.Insert(log); err != nil {
		logx.Errorf("[GostMonitor] 保存检查日志失败: %v", err)
	}
}

// triggerAlerts 根据检查结果触发告警
func triggerAlerts(r GostCheckResult) {
	if r.Status == "down" {
		triggerByType(entity.AlertTypeGostDown, entity.AlertLevelCritical,
			r.ServerId, r.ServerName,
			fmt.Sprintf("GOST 服务不可达: %s (%s)", r.ServerName, r.ServerHost),
			map[string]string{"error": r.ErrorMessage, "host": r.ServerHost})
	}

	if len(r.MissingPorts) > 0 {
		triggerByType(entity.AlertTypeGostForwardMissing, entity.AlertLevelWarning,
			r.ServerId, r.ServerName,
			fmt.Sprintf("GOST 转发规则不完整: %s 期望%d个端口, 实际%d个", r.ServerName, r.ExpectedPorts, r.ActualPorts),
			map[string]interface{}{"missing_ports": r.MissingPorts, "expected": r.ExpectedPorts, "actual": r.ActualPorts})
	}

	if r.ErrorRate >= errorThreshold && r.TotalConns > 100 {
		triggerByType(entity.AlertTypeGostHighErrors, entity.AlertLevelError,
			r.ServerId, r.ServerName,
			fmt.Sprintf("GOST 错误率过高: %s 错误率=%.2f%%", r.ServerName, r.ErrorRate*100),
			map[string]interface{}{"error_rate": r.ErrorRate, "total_errors": r.TotalErrors, "total_conns": r.TotalConns})
	}
}

func triggerByType(alertType, level string, serverId int, serverName, message string, detail interface{}) {
	var rules []entity.AlertRules
	err := dbs.DBAdmin.Where("type = ? AND status = 1", alertType).Find(&rules)
	if err != nil {
		logx.Errorf("[GostMonitor] 查询告警规则失败: %v", err)
		return
	}
	for _, rule := range rules {
		alert.TriggerAlert(&rule, level, "server", serverId, serverName, message, detail)
	}
}

// QueryMonitorLogsReq 查询监控日志请求
type QueryMonitorLogsReq struct {
	Page     int    `form:"page" json:"page"`
	Size     int    `form:"size" json:"size"`
	ServerId int    `form:"server_id" json:"server_id"`
	Status   string `form:"status" json:"status"`
}

// GetMonitorLogs 查询历史监控日志
func GetMonitorLogs(req QueryMonitorLogsReq) ([]entity.GostMonitorLogs, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if req.ServerId > 0 {
		session.Where("server_id = ?", req.ServerId)
	}
	if req.Status != "" {
		session.Where("status = ?", req.Status)
	}

	var logs []entity.GostMonitorLogs
	total, err := session.OrderBy("created_at DESC").
		Limit(req.Size, (req.Page-1)*req.Size).
		FindAndCount(&logs)
	if err != nil {
		return nil, 0, fmt.Errorf("查询监控日志失败: %v", err)
	}

	return logs, total, nil
}
