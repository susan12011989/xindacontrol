package cloud_monitor

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	cloudMonitorService "server/internal/server/service/cloud_monitor"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes 统一云监控路由
func Routes(ge gin.IRouter) {
	group := ge.Group("/cloud/monitor", middleware.Authorization)
	group.GET("/metrics", getCloudMonitorMetrics)
}

func getCloudMonitorMetrics(c *gin.Context) {
	var req model.CloudMonitorMetricsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	data, err := cloudMonitorService.GetCloudMonitorMetrics(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}
