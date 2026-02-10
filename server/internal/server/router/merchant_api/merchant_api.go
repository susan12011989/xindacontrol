package merchant_api

import (
	"strings"

	"server/internal/dbhelper"
	"server/internal/server/middleware"
	featureService "server/internal/server/service/feature"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

// Routes 注册商户API路由（供商户服务调用，使用Basic Auth认证）
func Routes(ge gin.IRouter) {
	group := ge.Group("merchant-api", middleware.BasicAuthForMerchantAPI)

	// 获取商户完整配置（套餐配置、全局配置、应用配置）
	group.GET("config", getConfig)

	// 获取功能开关配置
	group.GET("features", getFeatures)

	// 检查单个功能是否启用
	group.GET("feature/:name", checkFeature)
}

// getConfig 根据请求IP获取商户配置
// 通过客户端IP识别商户，返回 control 数据库中存储的商户配置
func getConfig(c *gin.Context) {
	// 获取客户端IP
	clientIP := getClientIP(c)
	if clientIP == "" {
		result.GResult(c, 400, nil, "无法获取客户端IP")
		return
	}

	logx.Infof("商户配置请求: clientIP=%s", clientIP)

	// 根据IP获取商户信息
	m, err := dbhelper.GetMerchantByServerIP(clientIP)
	if err != nil {
		logx.Errorf("根据IP查询商户失败: ip=%s, err=%v", clientIP, err)
		result.GResult(c, 404, nil, err.Error())
		return
	}
	logx.Infof("商户配置请求成功返回: clientIP=%s data=%v", clientIP, m)
	// 返回 control 数据库中存储的配置
	result.GOK(c, gin.H{
		"merchant_no":           m.No,
		"name":                  m.Name,
		"status":                m.Status,
		"expired_at":            m.ExpiredAt.Unix(),
		"package_configuration": m.PackageConfiguration,
		"configs":               m.Configs,
		"app_configs":           m.AppConfigs,
	})
}

// getFeatures 获取商户的所有功能开关
func getFeatures(c *gin.Context) {
	clientIP := getClientIP(c)
	if clientIP == "" {
		result.GResult(c, 400, nil, "无法获取客户端IP")
		return
	}

	// 根据IP获取商户信息
	m, err := dbhelper.GetMerchantByServerIP(clientIP)
	if err != nil {
		logx.Errorf("根据IP查询商户失败: ip=%s, err=%v", clientIP, err)
		result.GResult(c, 404, nil, err.Error())
		return
	}

	// 获取商户的功能开关配置
	flags, err := featureService.GetFeatureFlagsMap(m.Id)
	if err != nil {
		logx.Errorf("获取功能开关失败: merchant_id=%d, err=%v", m.Id, err)
		result.GResult(c, 500, nil, "获取功能开关失败")
		return
	}

	result.GOK(c, gin.H{
		"merchant_id": m.Id,
		"merchant_no": m.No,
		"features":    flags,
	})
}

// checkFeature 检查单个功能是否启用
func checkFeature(c *gin.Context) {
	featureName := c.Param("name")
	if featureName == "" {
		result.GResult(c, 400, nil, "功能名称不能为空")
		return
	}

	clientIP := getClientIP(c)
	if clientIP == "" {
		result.GResult(c, 400, nil, "无法获取客户端IP")
		return
	}

	// 根据IP获取商户信息
	m, err := dbhelper.GetMerchantByServerIP(clientIP)
	if err != nil {
		logx.Errorf("根据IP查询商户失败: ip=%s, err=%v", clientIP, err)
		result.GResult(c, 404, nil, err.Error())
		return
	}

	// 检查功能是否启用
	enabled, err := featureService.CheckFeatureEnabled(m.Id, featureName)
	if err != nil {
		logx.Errorf("检查功能开关失败: merchant_id=%d, feature=%s, err=%v", m.Id, featureName, err)
		result.GResult(c, 500, nil, "检查功能开关失败")
		return
	}

	result.GOK(c, gin.H{
		"merchant_id":  m.Id,
		"feature_name": featureName,
		"enabled":      enabled,
	})
}

// getClientIP 获取客户端真实IP
func getClientIP(c *gin.Context) string {
	// 优先从 X-Forwarded-For 获取（可能经过代理）
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// 其次从 X-Real-IP 获取
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// 最后从 RemoteAddr 获取
	ip := c.ClientIP()
	return ip
}
