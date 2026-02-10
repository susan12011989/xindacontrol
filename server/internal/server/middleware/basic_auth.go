package middleware

import (
	"server/internal/server/cfg"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// BasicAuthForMerchantAPI 商户API的Basic Auth中间件
func BasicAuthForMerchantAPI(ctx *gin.Context) {
	user, pass, hasAuth := ctx.Request.BasicAuth()

	if !hasAuth {
		ctx.Header("WWW-Authenticate", `Basic realm="Merchant API"`)
		result.UnAuthorization(ctx, "缺少认证信息")
		ctx.Abort()
		return
	}

	// 从配置读取认证信息
	if cfg.C.MerchantAPI == nil ||
		user != cfg.C.MerchantAPI.Username ||
		pass != cfg.C.MerchantAPI.Password {
		result.UnAuthorization(ctx, "用户名或密码错误")
		ctx.Abort()
		return
	}

	ctx.Set("api_user", user)
	ctx.Next()
}
