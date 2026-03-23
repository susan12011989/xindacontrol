package wukongim

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	wukongimService "server/internal/server/service/wukongim"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes WuKongIM监控管理路由注册
func Routes(gi gin.IRouter) {
	group := gi.Group("wukongim")
	group.Use(middleware.Authorization)

	// 服务器状态
	group.GET("varz", getVarz)     // 系统变量（连接数/CPU/内存/消息量）
	group.GET("connz", getConnz)   // 连接详情（支持过滤/排序/分页）

	// 用户管理
	group.POST("user/onlinestatus", getOnlineStatus) // 用户在线状态
	group.POST("user/device_quit", deviceQuit)        // 强制下线
}

// getVarz 获取WuKongIM系统变量
func getVarz(ctx *gin.Context) {
	var req model.WuKongIMBaseReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := wukongimService.GetVarz(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getConnz 获取WuKongIM连接信息
func getConnz(ctx *gin.Context) {
	var req model.WuKongIMConnzReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}

	data, err := wukongimService.GetConnz(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getOnlineStatus 查询用户在线状态
func getOnlineStatus(ctx *gin.Context) {
	var req model.WuKongIMOnlineStatusReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := wukongimService.GetOnlineStatus(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// deviceQuit 强制设备下线
func deviceQuit(ctx *gin.Context) {
	var req model.WuKongIMDeviceQuitReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	if err := wukongimService.DeviceQuit(req); err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}
