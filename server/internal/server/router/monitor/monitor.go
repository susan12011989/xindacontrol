package monitor

import (
	"server/internal/server/middleware"
	monitorSvc "server/internal/server/service/monitor"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes GOST 监控路由
func Routes(ge gin.IRouter) {
	group := ge.Group("monitor", middleware.Authorization)

	// GOST 健康检查
	group.POST("gost/check", checkGostServers)
	// 历史检查记录
	group.GET("gost/logs", getMonitorLogs)
}

// checkGostServers 手动触发 GOST 服务器健康检查
func checkGostServers(c *gin.Context) {
	results, err := monitorSvc.CheckAllGostServers()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": results})
}

// getMonitorLogs 查询历史监控日志
func getMonitorLogs(c *gin.Context) {
	var req monitorSvc.QueryMonitorLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	list, total, err := monitorSvc.GetMonitorLogs(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{"list": list, "total": total})
}
