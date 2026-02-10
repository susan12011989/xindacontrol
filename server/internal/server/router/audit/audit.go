package audit

import (
	"server/internal/server/middleware"
	auditSvc "server/internal/server/service/audit"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes 审计日志路由
func Routes(ge gin.IRouter) {
	group := ge.Group("audit", middleware.Authorization)
	group.GET("logs", listAuditLogs)
	group.GET("action-options", getActionOptions)
	group.GET("target-type-options", getTargetTypeOptions)
}

// listAuditLogs 查询审计日志列表
func listAuditLogs(c *gin.Context) {
	var req auditSvc.QueryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	list, total, err := auditSvc.Query(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// getActionOptions 获取操作类型选项
func getActionOptions(c *gin.Context) {
	result.GOK(c, auditSvc.GetActionOptions())
}

// getTargetTypeOptions 获取目标类型选项
func getTargetTypeOptions(c *gin.Context) {
	result.GOK(c, auditSvc.GetTargetTypeOptions())
}
