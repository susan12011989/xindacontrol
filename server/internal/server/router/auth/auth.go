package auth

import (
	"server/internal/dbhelper"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/internal/server/service/auth"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

func Routes(gi gin.IRouter) {
	group := gi.Group("auth")
	group.GET("challenge", challenge)
	group.POST("login", login)
	group.GET("me", middleware.Authorization, me)

	// 2FA
	group2fa := gi.Group("2fa")
	group2fa.Use(middleware.Authorization)

	group2fa.GET("status", getTwoFAStatus)
	group2fa.GET("setup", getTwoFASetup)
	group2fa.POST("enable", enableTwoFA)
	group2fa.POST("disable", disableTwoFA)
}

func login(ctx *gin.Context) {
	var encReq model.EncryptedLoginReq
	err := ctx.ShouldBindJSON(&encReq)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}
	// 将 UA 透传到服务层（在解密路径中继续透传）
	data, err := auth.LoginEncryptedWithUA(ctx.ClientIP(), ctx.GetHeader("User-Agent"), encReq)
	if err != nil {
		result.GResult(ctx, 400, nil, err.Error())
		return
	}
	result.GOK(ctx, data)
}

func me(ctx *gin.Context) {
	username := middleware.GetUsername(ctx)
	role := "user"
	if username != "" {
		if user, err := dbhelper.GetSysUserByUsername(username); err == nil {
			if user.Role != "" {
				role = user.Role
			}
		}
	}

	result.GOK(ctx, model.MeResp{
		Username:         username,
		Roles:            []string{role},
		TwoFactorEnabled: middleware.GetTwoFA(ctx),
		Ip:               ctx.ClientIP(),
	})
}

func challenge(ctx *gin.Context) {
	data, err := auth.GetChallenge()
	if err != nil {
		result.GResult(ctx, 500, nil, err.Error())
		return
	}
	result.GOK(ctx, data)
}
