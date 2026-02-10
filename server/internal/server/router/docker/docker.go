package docker

import (
	"fmt"
	"server/internal/server/middleware"
	"server/internal/server/model"
	dockerService "server/internal/server/service/docker"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes Docker容器管理路由注册
func Routes(gi gin.IRouter) {
	group := gi.Group("docker")
	group.Use(middleware.Authorization) // 需要认证

	// 容器列表与状态
	group.GET("containers", listContainers)           // 获取容器列表
	group.GET("containers/stats", getContainersStats) // 获取资源使用

	// 容器日志
	group.GET("logs", getContainerLogs) // 获取容器日志

	// 容器操作
	group.POST("containers/operate", operateContainer)   // 单个操作
	group.POST("containers/batch-operate", batchOperate) // 批量操作

	// 操作历史
	group.GET("history", queryHistory) // 查询操作历史

	// 服务器健康检查
	group.GET("health", checkHealth)            // 单个服务器健康检查
	group.POST("health/batch", batchCheckHealth) // 批量健康检查
}

// listContainers 获取容器列表
func listContainers(ctx *gin.Context) {
	var req model.QueryDockerContainersReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := dockerService.QueryContainers(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getContainersStats 获取容器资源使用情况
func getContainersStats(ctx *gin.Context) {
	serverIdStr := ctx.Query("server_id")
	if serverIdStr == "" {
		result.GResult(ctx, 601, nil, "server_id参数不能为空")
		return
	}

	serverId := 0
	if _, err := fmt.Sscanf(serverIdStr, "%d", &serverId); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := dockerService.GetContainerStats(serverId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{
		"list": data,
	})
}

// getContainerLogs 获取容器日志
func getContainerLogs(ctx *gin.Context) {
	var req model.GetDockerLogsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := dockerService.GetContainerLogs(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// operateContainer 操作容器
func operateContainer(ctx *gin.Context) {
	var req model.DockerContainerOperationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 获取操作人
	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := dockerService.OperateContainer(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// batchOperate 批量操作容器
func batchOperate(ctx *gin.Context) {
	var req model.DockerBatchOperationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 获取操作人
	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := dockerService.BatchOperateContainers(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// queryHistory 查询Docker操作历史
func queryHistory(ctx *gin.Context) {
	var req model.QueryDockerHistoryReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 设置默认分页
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	data, err := dockerService.QueryDockerHistory(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// checkHealth 单个服务器健康检查
func checkHealth(ctx *gin.Context) {
	var req model.HealthCheckReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := dockerService.CheckServerHealth(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// batchCheckHealth 批量健康检查
func batchCheckHealth(ctx *gin.Context) {
	var req model.BatchHealthCheckReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := dockerService.BatchCheckServerHealth(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}
