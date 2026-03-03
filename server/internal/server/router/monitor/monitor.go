package monitor

import (
	"fmt"
	"server/internal/server/middleware"
	"server/internal/server/model"
	monitorSvc "server/internal/server/service/monitor"
	"server/pkg/result"
	"time"

	"github.com/gin-gonic/gin"
)

// Routes GOST 监控路由
func Routes(ge gin.IRouter) {
	group := ge.Group("monitor", middleware.Authorization)

	// GOST 健康检查
	group.POST("gost/check", checkGostServers)
	// 历史检查记录
	group.GET("gost/logs", getMonitorLogs)
	// 带宽测速（流式API）
	group.POST("gost/bandwidth-test", bandwidthTest)
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

// bandwidthTest 带宽测速（流式API）
func bandwidthTest(c *gin.Context) {
	var req model.BandwidthTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}

	result.GStream(c)

	testResult, err := monitorSvc.RunBandwidthTest(req.ServerId, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}

	// 最后一条消息附带结构化结果
	result.GStreamData(c, gin.H{
		"result": testResult,
	})
	result.GStreamEnd(c, true, "测速完成")
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
