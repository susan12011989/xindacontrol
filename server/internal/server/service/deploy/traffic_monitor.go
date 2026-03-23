package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== 隧道连接数采集 ====================

// GetTunnelStats 获取单台 GOST 服务器的隧道连接统计（实时）
func GetTunnelStats(req model.GetTrafficStatsReq) (model.TunnelStatsResp, error) {
	var resp model.TunnelStatsResp

	// 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %w", err)
	}
	if !has {
		return resp, fmt.Errorf("服务器不存在")
	}

	// 通过 GOST API 获取隧道统计
	tunnels, err := gostapi.GetServiceStats(server.Host)
	if err != nil {
		return resp, fmt.Errorf("获取GOST统计失败: %w", err)
	}

	for _, t := range tunnels {
		item := model.TunnelStatItem{
			Name:         t.Name,
			Port:         t.Port,
			Target:       t.Target,
			CurrentConns: int(t.CurrentConns),
			TotalConns:   int64(t.TotalConns),
			InputBytes:   int64(t.InputBytes),
			OutputBytes:  int64(t.OutputBytes),
			TotalErrs:    int64(t.TotalErrs),
			State:        t.State,
		}
		resp.Tunnels = append(resp.Tunnels, item)
		resp.TotalCurrentConns += item.CurrentConns
		resp.TotalTunnels++
		if item.CurrentConns == 0 {
			resp.IdleTunnels++
		}
	}

	// 告警判断
	resp.AlertLevel, resp.AlertMsg = evaluateTunnelAlert(resp)

	return resp, nil
}

// GetTunnelStatsBatch 批量获取多台 GOST 服务器的隧道统计
func GetTunnelStatsBatch(req model.GetTrafficStatsBatchReq) (model.TunnelStatsBatchResp, error) {
	var resp model.TunnelStatsBatchResp
	if len(req.ServerIds) == 0 {
		return resp, nil
	}

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

			stat := model.ServerTunnelSummary{ServerId: serverID}
			tunnelResp, err := GetTunnelStats(model.GetTrafficStatsReq{ServerId: serverID})
			if err != nil {
				stat.Error = err.Error()
			} else {
				stat.TotalTunnels = tunnelResp.TotalTunnels
				stat.TotalCurrentConns = tunnelResp.TotalCurrentConns
				stat.IdleTunnels = tunnelResp.IdleTunnels
				stat.AlertLevel = tunnelResp.AlertLevel
				stat.AlertMsg = tunnelResp.AlertMsg
			}

			mu.Lock()
			resp.Stats = append(resp.Stats, stat)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return resp, nil
}

// ==================== 定期采集 & 存储 ====================

// CollectAndSaveTunnelStats 采集所有 GOST 服务器的隧道统计并存入数据库
// 由定时任务调用
func CollectAndSaveTunnelStats() {
	logx.Info("[隧道监控] 开始定期采集...")

	// 查所有启用的系统服务器（GOST 部署在系统服务器上）
	var servers []entity.Servers
	err := dbs.DBAdmin.Where("status = 1").Find(&servers)
	if err != nil {
		logx.Errorf("[隧道监控] 查询服务器列表失败: %v", err)
		return
	}

	now := time.Now()
	var records []entity.TunnelStatsRecord

	for _, server := range servers {
		tunnels, err := gostapi.GetServiceStats(server.Host)
		if err != nil {
			logx.Errorf("[隧道监控] 采集服务器 %s(%d) 失败: %v", server.Host, server.Id, err)
			continue
		}

		// 查关联的商户信息
		merchantName := ""
		merchantId := 0
		if server.MerchantId > 0 {
			var merchant entity.Merchants
			has, _ := dbs.DBAdmin.Where("id = ?", server.MerchantId).Get(&merchant)
			if has {
				merchantName = merchant.Name
				merchantId = int(merchant.Id)
			}
		}

		for _, t := range tunnels {
			records = append(records, entity.TunnelStatsRecord{
				ServerId:     int(server.Id),
				ServerHost:   server.Host,
				MerchantId:   merchantId,
				MerchantName: merchantName,
				TunnelName:   t.Name,
				Port:         t.Port,
				Target:       t.Target,
				CurrentConns: int(t.CurrentConns),
				TotalConns:   int64(t.TotalConns),
				InputBytes:   int64(t.InputBytes),
				OutputBytes:  int64(t.OutputBytes),
				TotalErrs:    int64(t.TotalErrs),
				State:        t.State,
				CollectedAt:  now,
			})
		}
	}

	if len(records) > 0 {
		_, err := dbs.DBAdmin.Insert(&records)
		if err != nil {
			logx.Errorf("[隧道监控] 写入统计记录失败: %v", err)
		} else {
			logx.Infof("[隧道监控] 采集完成，写入 %d 条记录", len(records))
		}
	}

	// 清理30天前的旧记录
	cleanBefore := now.AddDate(0, 0, -30)
	affected, err := dbs.DBAdmin.Where("collected_at < ?", cleanBefore).Delete(&entity.TunnelStatsRecord{})
	if err != nil {
		logx.Errorf("[隧道监控] 清理旧记录失败: %v", err)
	} else if affected > 0 {
		logx.Infof("[隧道监控] 清理 %d 条30天前的旧记录", affected)
	}
}

// GetTunnelStatsHistory 查询隧道统计历史记录
func GetTunnelStatsHistory(req model.TunnelStatsHistoryReq) (model.TunnelStatsHistoryResp, error) {
	var resp model.TunnelStatsHistoryResp

	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if req.ServerId > 0 {
		session = session.Where("server_id = ?", req.ServerId)
	}
	if req.MerchantId > 0 {
		session = session.Where("merchant_id = ?", req.MerchantId)
	}
	if req.TunnelName != "" {
		session = session.Where("tunnel_name = ?", req.TunnelName)
	}
	if req.StartTime != "" {
		session = session.Where("collected_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		session = session.Where("collected_at <= ?", req.EndTime)
	}

	var records []entity.TunnelStatsRecord
	count, err := session.OrderBy("collected_at DESC").
		Limit(req.PageSize, (req.Page-1)*req.PageSize).
		FindAndCount(&records)
	if err != nil {
		return resp, fmt.Errorf("查询历史记录失败: %w", err)
	}

	resp.Total = int(count)
	for _, r := range records {
		resp.Records = append(resp.Records, model.TunnelStatsHistoryItem{
			ServerId:     r.ServerId,
			ServerHost:   r.ServerHost,
			MerchantId:   r.MerchantId,
			MerchantName: r.MerchantName,
			TunnelName:   r.TunnelName,
			Port:         r.Port,
			Target:       r.Target,
			CurrentConns: r.CurrentConns,
			TotalConns:   r.TotalConns,
			InputBytes:   r.InputBytes,
			OutputBytes:  r.OutputBytes,
			TotalErrs:    r.TotalErrs,
			State:        r.State,
			CollectedAt:  r.CollectedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, nil
}

// ==================== 商户汇总视图 ====================

// GetMerchantTunnelOverview 获取所有商户的隧道连接汇总
// 表格：商户 | IP | 转发状态 | 连接数
func GetMerchantTunnelOverview() (model.MerchantTunnelOverviewResp, error) {
	var resp model.MerchantTunnelOverviewResp

	// 查所有有 GOST 服务的服务器（关联商户）
	var servers []entity.Servers
	err := dbs.DBAdmin.Where("status = 1").Find(&servers)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %w", err)
	}

	// 获取所有商户信息
	merchantMap := make(map[int]string)
	var merchants []entity.Merchants
	_ = dbs.DBAdmin.Find(&merchants)
	for _, m := range merchants {
		merchantMap[int(m.Id)] = m.Name
	}

	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, server := range servers {
		srv := server
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			summary := model.MerchantTunnelSummary{
				ServerId:     int(srv.Id),
				ServerIP:     srv.Host,
				MerchantId:   srv.MerchantId,
				MerchantName: merchantMap[srv.MerchantId],
			}
			if summary.MerchantName == "" {
				summary.MerchantName = fmt.Sprintf("服务器#%d", srv.Id)
			}

			// 通过 GOST API 获取隧道统计
			tunnels, err := gostapi.GetServiceStats(srv.Host)
			if err != nil {
				summary.ForwardState = "offline"
				summary.AlertLevel = "danger"
			} else {
				allIdle := true
				for _, t := range tunnels {
					summary.TunnelCount++
					summary.CurrentConns += int(t.CurrentConns)
					summary.TotalConns += int64(t.TotalConns)
					summary.ErrorCount += int64(t.TotalErrs)
					if t.CurrentConns == 0 {
						summary.IdleCount++
					} else {
						allIdle = false
					}
				}

				if summary.TunnelCount == 0 {
					summary.ForwardState = "offline"
				} else if allIdle {
					summary.ForwardState = "idle"
				} else if summary.IdleCount > 0 {
					summary.ForwardState = "partial"
				} else {
					summary.ForwardState = "online"
				}

				// 告警
				if summary.CurrentConns > dangerConnsPerTunnel {
					summary.AlertLevel = "danger"
				} else if summary.CurrentConns > warningConnsPerTunnel {
					summary.AlertLevel = "warning"
				} else {
					summary.AlertLevel = "normal"
				}
			}

			mu.Lock()
			resp.Merchants = append(resp.Merchants, summary)
			mu.Unlock()
		}()
	}

	wg.Wait()
	resp.Total = len(resp.Merchants)
	return resp, nil
}

// ==================== 聚合统计（按时间区间） ====================

// GetTunnelStatsAggregate 按时间粒度聚合统计隧道连接数
// 支持按小时/天聚合，可按商户/服务器筛选
func GetTunnelStatsAggregate(req model.TunnelStatsAggregateReq) (model.TunnelStatsAggregateResp, error) {
	var resp model.TunnelStatsAggregateResp

	groupBy := req.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}

	// 时间格式化 SQL
	var timeFormat string
	switch groupBy {
	case "hour":
		timeFormat = "%Y-%m-%d %H:00"
	default:
		timeFormat = "%Y-%m-%d"
	}

	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	// 构建查询
	sqlStr := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(collected_at, '%s') as time_slot,
			merchant_name,
			server_host,
			MAX(current_conns) as max_conns,
			ROUND(AVG(current_conns)) as avg_conns,
			MAX(total_conns) as total_conns,
			MAX(input_bytes) as total_input,
			MAX(output_bytes) as total_output,
			MAX(total_errs) as total_errs,
			COUNT(*) as sample_count
		FROM tunnel_stats_record
		WHERE 1=1
	`, timeFormat)

	args := make([]interface{}, 0)

	if req.MerchantId > 0 {
		sqlStr += " AND merchant_id = ?"
		args = append(args, req.MerchantId)
	}
	if req.ServerId > 0 {
		sqlStr += " AND server_id = ?"
		args = append(args, req.ServerId)
	}
	if req.StartTime != "" {
		sqlStr += " AND collected_at >= ?"
		args = append(args, req.StartTime)
	}
	if req.EndTime != "" {
		sqlStr += " AND collected_at <= ?"
		args = append(args, req.EndTime+" 23:59:59")
	}

	sqlStr += fmt.Sprintf(" GROUP BY time_slot, merchant_name, server_host ORDER BY time_slot DESC")

	results, err := session.QueryString(append([]interface{}{sqlStr}, args...)...)
	if err != nil {
		return resp, fmt.Errorf("聚合查询失败: %w", err)
	}

	for _, row := range results {
		item := model.TunnelStatsAggregateItem{
			TimeSlot:     row["time_slot"],
			MerchantName: row["merchant_name"],
			ServerIP:     row["server_host"],
		}
		fmt.Sscanf(row["max_conns"], "%d", &item.MaxConns)
		fmt.Sscanf(row["avg_conns"], "%d", &item.AvgConns)
		fmt.Sscanf(row["total_conns"], "%d", &item.TotalConns)
		fmt.Sscanf(row["total_input"], "%d", &item.TotalInput)
		fmt.Sscanf(row["total_output"], "%d", &item.TotalOutput)
		fmt.Sscanf(row["total_errs"], "%d", &item.TotalErrs)
		fmt.Sscanf(row["sample_count"], "%d", &item.SampleCount)
		resp.Items = append(resp.Items, item)
	}

	resp.Total = len(resp.Items)
	return resp, nil
}

// ==================== 告警判断 ====================

const (
	warningConnsPerTunnel = 500  // 单隧道连接数告警
	dangerConnsPerTunnel  = 2000 // 单隧道连接数危险
	warningTotalConns     = 5000 // 总连接数告警
)

func evaluateTunnelAlert(stats model.TunnelStatsResp) (level, msg string) {
	level = "normal"
	var alerts []string

	// 检查单隧道连接数
	for _, t := range stats.Tunnels {
		if t.CurrentConns > dangerConnsPerTunnel {
			level = "danger"
			alerts = append(alerts, fmt.Sprintf("隧道 %s 连接数异常：%d", t.Name, t.CurrentConns))
		} else if t.CurrentConns > warningConnsPerTunnel {
			if level != "danger" {
				level = "warning"
			}
			alerts = append(alerts, fmt.Sprintf("隧道 %s 连接数偏高：%d", t.Name, t.CurrentConns))
		}
	}

	// 检查总连接数
	if stats.TotalCurrentConns > warningTotalConns {
		if level != "danger" {
			level = "warning"
		}
		alerts = append(alerts, fmt.Sprintf("总连接数偏高：%d", stats.TotalCurrentConns))
	}

	if len(alerts) > 0 {
		msg = alerts[0] // 只取第一条
		if len(alerts) > 1 {
			msg += fmt.Sprintf("（共%d项告警）", len(alerts))
		}
	}

	return
}

// ==================== 应急响应（仅超管） ====================

// BlockIP 封禁指定 IP（通过 SSH 执行 iptables）
func BlockIP(req model.BlockIPReq) error {
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return err
	}

	checkCmd := fmt.Sprintf("iptables -C INPUT -s %s -j DROP 2>/dev/null && echo EXISTS || echo NOT", req.IP)
	output := client.SSHClient.ExecuteCommandSilent(checkCmd)
	if output == "EXISTS" {
		return fmt.Errorf("IP %s 已在封禁列表中", req.IP)
	}

	blockCmd := fmt.Sprintf("iptables -I INPUT -s %s -j DROP", req.IP)
	if _, err := client.SSHClient.ExecuteCommand(blockCmd); err != nil {
		return fmt.Errorf("封禁失败: %v", err)
	}

	if req.Duration != "" && req.Duration != "permanent" {
		unblockCmd := fmt.Sprintf("(sleep %s && iptables -D INPUT -s %s -j DROP) &", parseDuration(req.Duration), req.IP)
		client.SSHClient.ExecuteCommandSilent(unblockCmd)
	}

	return nil
}

// UnblockIP 解封 IP
func UnblockIP(serverId int, ip string) error {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("iptables -D INPUT -s %s -j DROP 2>/dev/null; echo OK", ip)
	_, err = client.SSHClient.ExecuteCommand(cmd)
	return err
}

// GetBlockedIPs 获取封禁列表
func GetBlockedIPs(serverId int) ([]string, error) {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return nil, err
	}
	cmd := "iptables -L INPUT -n 2>/dev/null | grep DROP | awk '{print $4}' | grep -E '^[0-9]+\\.' | sort -u"
	output := client.SSHClient.ExecuteCommandSilent(cmd)
	if output == "" {
		return []string{}, nil
	}
	var ips []string
	for _, line := range splitLines(output) {
		if line != "" && line != "0.0.0.0/0" {
			ips = append(ips, line)
		}
	}
	return ips, nil
}

// EmergencyRateLimit 紧急限流
func EmergencyRateLimit(req model.EmergencyRateLimitReq) error {
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return err
	}
	ssh := client.SSHClient

	// 清除旧规则
	ssh.ExecuteCommandSilent("iptables -F TSDD_RATELIMIT 2>/dev/null")
	ssh.ExecuteCommandSilent("iptables -D INPUT -j TSDD_RATELIMIT 2>/dev/null")
	ssh.ExecuteCommandSilent("iptables -X TSDD_RATELIMIT 2>/dev/null")

	if req.MaxConnPerIP == 0 && req.MaxSynRate == 0 {
		return nil // 取消限流
	}

	ssh.ExecuteCommandSilent("iptables -N TSDD_RATELIMIT 2>/dev/null")

	if req.MaxConnPerIP > 0 {
		cmd := fmt.Sprintf("iptables -A TSDD_RATELIMIT -p tcp --syn -m connlimit --connlimit-above %d -j DROP", req.MaxConnPerIP)
		if _, err := ssh.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("设置单IP连接限制失败: %v", err)
		}
	}

	if req.MaxSynRate > 0 {
		cmd := fmt.Sprintf("iptables -A TSDD_RATELIMIT -p tcp --syn -m limit --limit %d/s --limit-burst %d -j ACCEPT", req.MaxSynRate, req.MaxSynRate*2)
		ssh.ExecuteCommandSilent(cmd)
		ssh.ExecuteCommandSilent("iptables -A TSDD_RATELIMIT -p tcp --syn -j DROP")
	}

	ssh.ExecuteCommandSilent("iptables -I INPUT -j TSDD_RATELIMIT")
	return nil
}

// ==================== 工具 ====================

func parseDuration(d string) string {
	d = strings.TrimSpace(d)
	if strings.HasSuffix(d, "h") {
		hours, _ := strconv.Atoi(strings.TrimSuffix(d, "h"))
		return fmt.Sprintf("%d", hours*3600)
	}
	if strings.HasSuffix(d, "m") {
		mins, _ := strconv.Atoi(strings.TrimSuffix(d, "m"))
		return fmt.Sprintf("%d", mins*60)
	}
	return "3600"
}

func splitLines(s string) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
