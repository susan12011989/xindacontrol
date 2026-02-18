package resource_overview

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	service "server/internal/server/service/resource_overview"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 资源总览路由注册
func Routes(gi gin.IRouter) {
	group := gi.Group("resource-overview")
	group.Use(middleware.Authorization)

	// 标签 CRUD
	group.GET("tags", listTags)
	group.POST("tags", createTag)
	group.PUT("tags/:id", updateTag)
	group.DELETE("tags/:id", deleteTag)

	// 标签分配
	group.POST("tags/assign", assignTags)
	group.POST("tags/remove", removeTags)

	// 全局资源列表
	group.GET("oss-configs", queryGlobalOssConfigs)
	group.GET("gost-servers", queryGlobalGostServers)

	// 批量操作
	group.POST("batch-sync-gost-ip", batchSyncGostIP)
	group.POST("check-oss-health", checkOssHealth)
}

// ========== 标签 CRUD ==========

func listTags(ctx *gin.Context) {
	data, err := service.ListTags()
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

func createTag(ctx *gin.Context) {
	var req model.ResourceTagReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	id, err := service.CreateTag(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, gin.H{"id": id})
}

func updateTag(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}
	var req model.ResourceTagReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	if err := service.UpdateTag(id, req); err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, nil)
}

func deleteTag(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}
	if err := service.DeleteTag(id); err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, nil)
}

// ========== 标签分配 ==========

func assignTags(ctx *gin.Context) {
	var req model.AssignTagsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	if err := service.AssignTags(req); err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, nil)
}

func removeTags(ctx *gin.Context) {
	var req model.RemoveTagsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	if err := service.RemoveTags(req); err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, nil)
}

// ========== 全局列表 ==========

func queryGlobalOssConfigs(ctx *gin.Context) {
	var req model.QueryGlobalOssConfigsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	data, err := service.QueryGlobalOssConfigs(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

func queryGlobalGostServers(ctx *gin.Context) {
	var req model.QueryGlobalGostServersReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	data, err := service.QueryGlobalGostServers(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// ========== 批量操作 ==========

func checkOssHealth(ctx *gin.Context) {
	var req model.CheckOssHealthReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	data := service.CheckOssHealth(req.OssConfigIds)
	result.GOK(ctx, data)
}

func batchSyncGostIP(ctx *gin.Context) {
	var req model.BatchSyncGostIPByFilterReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	data, err := service.BatchSyncGostIPByFilter(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}
