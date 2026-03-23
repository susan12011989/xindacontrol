package cron

import (
	"context"
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// GostMonitor GOST 链路健康监控
// 定时检测所有商户服务器的 GOST 状态，发现异常自动重启
type GostMonitor struct {
	interval time.Duration
	stopCh   chan struct{}
}

// NewGostMonitor 创建 GOST 监控器
func NewGostMonitor(interval time.Duration) *GostMonitor {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	return &GostMonitor{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动后台监控
func (m *GostMonitor) Start() {
	logx.Infof("[GostMonitor] 启动 GOST 链路监控，间隔 %v", m.interval)
	go m.run()
}

// Stop 停止监控
func (m *GostMonitor) Stop() {
	close(m.stopCh)
}

func (m *GostMonitor) run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAll()
		case <-m.stopCh:
			logx.Info("[GostMonitor] 监控已停止")
			return
		}
	}
}

func (m *GostMonitor) checkAll() {
	// 查询所有活跃的商户服务器（type=1 商户服务器）
	var servers []entity.Servers
	if err := dbs.DBAdmin.Where("status = 1 AND server_type = 1").Find(&servers); err != nil {
		logx.Errorf("[GostMonitor] 查询服务器失败: %v", err)
		return
	}

	for _, server := range servers {
		m.checkServer(server)
	}

	// 同时检查系统服务器上的 GOST
	var sysServers []entity.Servers
	if err := dbs.DBAdmin.Where("status = 1 AND server_type = 2").Find(&sysServers); err != nil {
		logx.Errorf("[GostMonitor] 查询系统服务器失败: %v", err)
		return
	}

	for _, server := range sysServers {
		m.checkServer(server)
	}
}

func (m *GostMonitor) checkServer(server entity.Servers) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = ctx // 预留给后续扩展

	// 尝试访问 GOST API
	_, err := gostapi.GetConfig(server.Host, "")
	if err != nil {
		logx.Errorf("[GostMonitor] 服务器 %s(%s) GOST API 不可达: %v", server.Name, server.Host, err)

		// 记录告警日志
		alertLog := entity.AlertLogs{
			RuleId:     0,
			Level:      "error",
			Type:       "gost_unreachable",
			TargetType: "server",
			TargetId:   server.Id,
			TargetName: server.Name,
			Message:    fmt.Sprintf("GOST 不可达: %s(%s)", server.Name, server.Host),
			Detail:     fmt.Sprintf("GOST API 无响应: %v", err),
			CreatedAt:  time.Now(),
		}
		if _, insertErr := dbs.DBAdmin.Insert(&alertLog); insertErr != nil {
			logx.Errorf("[GostMonitor] 记录告警失败: %v", insertErr)
		}
		return
	}

	// GOST API 正常，检查服务数量（0 个服务可能是配置丢失）
	serviceList, err := gostapi.GetServiceList(server.Host)
	if err != nil {
		return
	}

	if serviceList.Count == 0 && server.ServerType == 1 {
		logx.Infof("[GostMonitor] 服务器 %s(%s) GOST 服务数为 0，可能配置丢失", server.Name, server.Host)

		alertLog := entity.AlertLogs{
			RuleId:     0,
			Level:      "warning",
			Type:       "gost_empty_config",
			TargetType: "server",
			TargetId:   server.Id,
			TargetName: server.Name,
			Message:    fmt.Sprintf("GOST 配置为空: %s(%s)", server.Name, server.Host),
			Detail:     "GOST 运行中但无任何服务配置，可能需要重新部署",
			CreatedAt:  time.Now(),
		}
		dbs.DBAdmin.Insert(&alertLog)
	}
}
