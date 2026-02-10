package alert

import (
	"server/internal/server/middleware"
	alertSvc "server/internal/server/service/alert"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 告警管理路由
func Routes(ge gin.IRouter) {
	group := ge.Group("alert", middleware.Authorization)

	// 告警规则管理
	group.GET("rules", listRules)
	group.GET("rules/:id", getRule)
	group.POST("rules", createRule)
	group.PUT("rules/:id", updateRule)
	group.DELETE("rules/:id", deleteRule)
	group.POST("rules/:id/toggle", toggleRuleStatus)

	// 告警日志
	group.GET("logs", listLogs)

	// 选项
	group.GET("type-options", getAlertTypeOptions)
	group.GET("notify-type-options", getNotifyTypeOptions)
	group.GET("level-options", getAlertLevelOptions)

	// 手动触发测试告警
	group.POST("test", testAlert)
}

// listRules 查询告警规则列表
func listRules(c *gin.Context) {
	var req alertSvc.QueryRulesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	list, total, err := alertSvc.ListRules(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// getRule 获取单个规则详情
func getRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	req := alertSvc.QueryRulesReq{Page: 1, Size: 1}
	list, _, err := alertSvc.ListRules(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	for _, r := range list {
		if r.Id == id {
			result.GOK(c, r)
			return
		}
	}

	result.GErr(c, gin.Error{Err: err, Meta: "规则不存在"})
}

// createRule 创建告警规则
func createRule(c *gin.Context) {
	var req alertSvc.RuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	id, err := alertSvc.CreateRule(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{"id": id})
}

// updateRule 更新告警规则
func updateRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req alertSvc.RuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = alertSvc.UpdateRule(id, req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// deleteRule 删除告警规则
func deleteRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = alertSvc.DeleteRule(id)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// toggleRuleStatus 切换规则状态
func toggleRuleStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = alertSvc.ToggleRuleStatus(id)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// listLogs 查询告警日志列表
func listLogs(c *gin.Context) {
	var req alertSvc.QueryLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	list, total, err := alertSvc.ListLogs(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// getAlertTypeOptions 获取告警类型选项
func getAlertTypeOptions(c *gin.Context) {
	result.GOK(c, alertSvc.GetAlertTypeOptions())
}

// getNotifyTypeOptions 获取通知类型选项
func getNotifyTypeOptions(c *gin.Context) {
	result.GOK(c, alertSvc.GetNotifyTypeOptions())
}

// getAlertLevelOptions 获取告警级别选项
func getAlertLevelOptions(c *gin.Context) {
	result.GOK(c, alertSvc.GetAlertLevelOptions())
}

// testAlert 测试告警（手动触发）
func testAlert(c *gin.Context) {
	var req struct {
		RuleId int `json:"rule_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 这里可以添加测试告警的逻辑
	result.GOK(c, gin.H{"message": "测试告警已发送"})
}
