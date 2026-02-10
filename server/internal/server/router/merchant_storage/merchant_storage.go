package merchant_storage

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	merchantStorageService "server/internal/server/service/merchant_storage"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 注册商户存储配置路由
func Routes(gi gin.IRouter) {
	group := gi.Group("merchant-storage")
	group.Use(middleware.Authorization)

	group.GET("", queryStorageConfigs)           // GET /merchant-storage - 查询配置列表
	group.GET(":id", getStorageConfig)           // GET /merchant-storage/:id - 获取配置详情
	group.POST("", createStorageConfig)          // POST /merchant-storage - 创建配置
	group.PUT(":id", updateStorageConfig)        // PUT /merchant-storage/:id - 更新配置
	group.DELETE(":id", deleteStorageConfig)     // DELETE /merchant-storage/:id - 删除配置
	group.POST("push", pushStorageConfig)        // POST /merchant-storage/push - 推送配置到商户服务器
	group.GET("types", getStorageTypes)          // GET /merchant-storage/types - 获取存储类型选项
}

// 查询存储配置列表
func queryStorageConfigs(ctx *gin.Context) {
	var req model.QueryMerchantStorageReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	data, err := merchantStorageService.QueryMerchantStorageConfigs(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 获取存储配置详情
func getStorageConfig(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := merchantStorageService.GetMerchantStorageDetail(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 创建存储配置
func createStorageConfig(ctx *gin.Context) {
	var req model.MerchantStorageReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	id, err := merchantStorageService.CreateMerchantStorageConfig(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"id": id})
}

// 更新存储配置
func updateStorageConfig(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	var req model.MerchantStorageReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = merchantStorageService.UpdateMerchantStorageConfig(id, req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// 删除存储配置
func deleteStorageConfig(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = merchantStorageService.DeleteMerchantStorageConfig(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// 推送存储配置到商户服务器
func pushStorageConfig(ctx *gin.Context) {
	var req model.PushStorageConfigReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 获取当前登录用户名
	username := middleware.GetUsername(ctx)

	data, err := merchantStorageService.PushStorageConfig(req, username)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 获取存储类型选项
func getStorageTypes(ctx *gin.Context) {
	options := merchantStorageService.GetStorageTypeOptions()
	result.GOK(ctx, options)
}
