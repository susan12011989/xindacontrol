package auth

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/internal/server/service/auth"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// getTwoFAStatus 获取2FA状态
func getTwoFAStatus(ctx *gin.Context) {
	username := middleware.GetUsername(ctx)
	data, err := auth.GetTwoFAStatus(username)
	if err != nil {
		result.GResult(ctx, 500, nil, err.Error())
		return
	}
	result.GOK(ctx, data)
}

// getTwoFASetup 获取2FA设置信息（二维码等）
func getTwoFASetup(ctx *gin.Context) {
	username := middleware.GetUsername(ctx)
	data, err := auth.GetTwoFASetupInfo(username)
	if err != nil {
		result.GResult(ctx, 500, nil, err.Error())
		return
	}
	result.GOK(ctx, data)
}

// enableTwoFA 启用2FA
func enableTwoFA(ctx *gin.Context) {
	var req model.TwoFASetupReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	username := middleware.GetUsername(ctx)
	err := auth.EnableTwoFA(username, req.Code)
	if err != nil {
		result.GResult(ctx, 400, nil, err.Error())
		return
	}

	result.GOK(ctx, gin.H{"message": "2FA已启用"})
}

// disableTwoFA 禁用2FA
func disableTwoFA(ctx *gin.Context) {
	var req model.TwoFADisableReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	username := middleware.GetUsername(ctx)
	err := auth.DisableTwoFA(username, req.Password)
	if err != nil {
		result.GResult(ctx, 400, nil, err.Error())
		return
	}

	result.GOK(ctx, gin.H{"message": "2FA已禁用"})
}
