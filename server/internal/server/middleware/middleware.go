package middleware

import (
	"fmt"
	"server/pkg/result"
	"server/pkg/token_manager"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ips []string // 后台可访问白名单IP
)

func SetIps(newIps []string) {
	ips = newIps
}

// IPWhiteList 检查请求IP是否在白名单中
func IPWhiteList(ctx *gin.Context) {
	clientIP := ctx.ClientIP()

	// 如果白名单为空，则不进行限制
	if len(ips) == 0 {
		ctx.Next()
		return
	}

	// 检查IP是否在白名单中
	for _, ip := range ips {
		if ip == clientIP {
			ctx.Next()
			return
		}
	}

	// IP不在白名单中，返回404
	logx.Errorf("IP %s 不在IP白名单中进行访问后台", clientIP)
	ctx.Status(404)
	ctx.Abort()
}

func Authorization(ctx *gin.Context) {
	parts := strings.Split(ctx.GetHeader("Authorization"), " ")
	if len(parts) != 2 || parts[1] == "" {
		result.UnAuthorization(ctx, "未授权，请重新登录")
		ctx.Abort()
		return
	}
	tokenInfo, err := token_manager.ValidateToken(parts[1])
	if err != nil {
		// 根据错误类型返回更明确原因：过期、被顶号、无效
		msg := "未授权，请重新登录"
		e := err.Error()
		if strings.Contains(e, "expired") || strings.Contains(e, "not found") {
			msg = "登录已过期，请重新登录"
		} else if strings.Contains(e, "revoked") {
			msg = "你的账号已在其他设备登录，本次登录已失效"
		} else if strings.Contains(e, "invalid") || strings.Contains(e, "unexpected signing method") || strings.Contains(e, "parse token") {
			msg = "无效的Token，请重新登录"
		}
		logx.Errorf("未授权, 错误信息: %s, token: %s", e, parts[1])
		result.UnAuthorization(ctx, msg)
		ctx.Abort()
		return
	}
	ctx.Set("uid", tokenInfo.UserID)
	ctx.Set("tid", tokenInfo.TokenID)
	ctx.Set("username", tokenInfo.Username)
	ctx.Set("prefix", tokenInfo.Prefix)
	ctx.Set("two_fa", tokenInfo.TwoFA)
}
func GetPrefix(ctx *gin.Context) string {
	return ctx.GetString("prefix")
}
func GetUid(ctx *gin.Context) int64 {
	return ctx.GetInt64("uid")
}

func GetTid(ctx *gin.Context) string {
	return ctx.GetString("tid")
}

func GetUsername(ctx *gin.Context) string {
	return ctx.GetString("username")
}

func GetTwoFA(ctx *gin.Context) bool {
	return ctx.GetBool("two_fa")
}

func LogRequest(ctx *gin.Context) {
	start := time.Now()
	path := ctx.Request.URL.Path
	raw := ctx.Request.URL.RawQuery

	ctx.Next()

	// 获取请求ID（如果存在）
	requestID := GetRequestID(ctx)
	reqIDStr := ""
	if requestID != "" {
		reqIDStr = fmt.Sprintf("[%s] ", requestID[:8]) // 只显示前8位
	}

	// 记录请求日志
	logx.Infof("[admin] %s%s %s %s %d %v",
		reqIDStr,
		ctx.ClientIP(),
		ctx.Request.Method,
		path+"?"+raw,
		ctx.Writer.Status(),
		time.Since(start),
	)
}
