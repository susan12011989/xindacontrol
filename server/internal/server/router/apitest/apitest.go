package apitest

import (
	"errors"
	"server/internal/server/middleware"
	"server/internal/server/model"
	apiTestService "server/internal/server/service/apitest"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 注册API测试路由
func Routes(group *gin.RouterGroup) {
	g := group.Group("/api-test")
	g.Use(middleware.Authorization)

	// API目录
	g.GET("/catalog", getCatalog)

	// 测试用例管理
	g.POST("/cases", createTestCase)
	g.PUT("/cases/:id", updateTestCase)
	g.DELETE("/cases/:id", deleteTestCase)
	g.GET("/cases", queryTestCases)

	// 运行测试
	g.POST("/run", runAPITest)
	g.POST("/run/case/:id", runTestCase)
	g.POST("/run/batch", batchTest)

	// 监控配置管理
	g.POST("/monitors", createMonitor)
	g.PUT("/monitors/:id", updateMonitor)
	g.DELETE("/monitors/:id", deleteMonitor)
	g.GET("/monitors", queryMonitors)
	g.GET("/monitors/:id/history", queryMonitorHistory)
}

// getCatalog 获取API目录
func getCatalog(c *gin.Context) {
	resp := apiTestService.GetAPICatalog()
	result.GOK(c, resp)
}

// createTestCase 创建测试用例
func createTestCase(c *gin.Context) {
	var req model.TestCaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	merchantIdStr := c.Query("merchant_id")
	merchantId, _ := strconv.Atoi(merchantIdStr)
	if merchantId == 0 {
		result.GParamErr(c, errors.New("商户ID不能为空"))
		return
	}

	if err := apiTestService.CreateTestCase(merchantId, req); err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// updateTestCase 更新测试用例
func updateTestCase(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		result.GParamErr(c, errors.New("ID不能为空"))
		return
	}

	var req model.TestCaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := apiTestService.UpdateTestCase(id, req); err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// deleteTestCase 删除测试用例
func deleteTestCase(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		result.GParamErr(c, errors.New("ID不能为空"))
		return
	}

	if err := apiTestService.DeleteTestCase(id); err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// queryTestCases 查询测试用例
func queryTestCases(c *gin.Context) {
	var req model.QueryTestCaseReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	resp, err := apiTestService.QueryTestCases(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, resp)
}

// runAPITest 运行API测试
func runAPITest(c *gin.Context) {
	var req model.RunAPITestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	resp, err := apiTestService.RunAPITest(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, resp)
}

// runTestCase 运行单个测试用例
func runTestCase(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		result.GParamErr(c, errors.New("ID不能为空"))
		return
	}

	merchantIdStr := c.Query("merchant_id")
	merchantId, _ := strconv.Atoi(merchantIdStr)
	if merchantId == 0 {
		result.GParamErr(c, errors.New("商户ID不能为空"))
		return
	}

	resp, err := apiTestService.RunTestCase(merchantId, id)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, resp)
}

// batchTest 批量测试
func batchTest(c *gin.Context) {
	var req model.BatchTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	resp, err := apiTestService.BatchTest(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, resp)
}

// createMonitor 创建监控配置
func createMonitor(c *gin.Context) {
	var req model.MonitorConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := apiTestService.CreateMonitorConfig(req); err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// updateMonitor 更新监控配置
func updateMonitor(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		result.GParamErr(c, errors.New("ID不能为空"))
		return
	}

	var req model.MonitorConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := apiTestService.UpdateMonitorConfig(id, req); err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// deleteMonitor 删除监控配置
func deleteMonitor(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		result.GParamErr(c, errors.New("ID不能为空"))
		return
	}

	if err := apiTestService.DeleteMonitorConfig(id); err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// queryMonitors 查询监控配置
func queryMonitors(c *gin.Context) {
	var req model.QueryMonitorReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	resp, err := apiTestService.QueryMonitorConfigs(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, resp)
}

// queryMonitorHistory 查询监控历史
func queryMonitorHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		result.GParamErr(c, errors.New("ID不能为空"))
		return
	}

	var req model.QueryMonitorHistoryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	req.MonitorId = id
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	resp, err := apiTestService.QueryMonitorHistory(req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, resp)
}
