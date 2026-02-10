package feature

import (
	"fmt"
	"server/internal/server/middleware"
	"server/internal/server/model"
	featureService "server/internal/server/service/feature"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes 功能开关路由注册
func Routes(gi gin.IRouter) {
	group := gi.Group("feature")
	group.Use(middleware.Authorization) // 需要认证

	// 获取功能定义列表
	group.GET("definitions", getFeatureDefinitions)

	// 商户功能开关管理
	group.GET("flags", getFeatureFlags)            // 获取商户功能开关
	group.PUT("flags", updateFeatureFlag)          // 更新单个功能开关
	group.PUT("flags/batch", batchUpdateFlags)     // 批量更新功能开关
	group.POST("flags/init", initFeatureFlags)     // 初始化商户功能开关

	// 检查功能状态（供外部调用）
	group.GET("check", checkFeature) // 检查某功能是否启用
}

// getFeatureDefinitions 获取所有可用功能定义
func getFeatureDefinitions(ctx *gin.Context) {
	definitions := featureService.GetAllFeatureDefinitions()
	result.GOK(ctx, gin.H{
		"list": definitions,
	})
}

// getFeatureFlags 获取商户功能开关列表
func getFeatureFlags(ctx *gin.Context) {
	var req model.QueryFeatureFlagsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := featureService.GetFeatureFlags(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// updateFeatureFlag 更新单个功能开关
func updateFeatureFlag(ctx *gin.Context) {
	var req model.UpdateFeatureFlagReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err := featureService.UpdateFeatureFlag(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, model.FeatureFlagOperationResponse{
		Success: true,
		Message: "更新成功",
	})
}

// batchUpdateFlags 批量更新功能开关
func batchUpdateFlags(ctx *gin.Context) {
	var req model.BatchUpdateFeatureFlagsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err := featureService.BatchUpdateFeatureFlags(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, model.FeatureFlagOperationResponse{
		Success: true,
		Message: "批量更新成功",
	})
}

// initFeatureFlags 初始化商户功能开关
func initFeatureFlags(ctx *gin.Context) {
	var req model.InitFeatureFlagsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err := featureService.InitFeatureFlags(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, model.FeatureFlagOperationResponse{
		Success: true,
		Message: "初始化成功",
	})
}

// checkFeature 检查功能是否启用
func checkFeature(ctx *gin.Context) {
	merchantIdStr := ctx.Query("merchant_id")
	featureName := ctx.Query("feature_name")

	if merchantIdStr == "" || featureName == "" {
		result.GResult(ctx, 601, nil, "merchant_id和feature_name参数不能为空")
		return
	}

	var merchantId int
	if _, err := fmt.Sscanf(merchantIdStr, "%d", &merchantId); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	enabled, err := featureService.CheckFeatureEnabled(merchantId, featureName)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{
		"merchant_id":  merchantId,
		"feature_name": featureName,
		"enabled":      enabled,
	})
}
