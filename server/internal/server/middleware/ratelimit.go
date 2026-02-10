package middleware

import (
	"context"
	"fmt"
	"net/http"
	"server/pkg/dbs"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 每个时间窗口允许的最大请求数
	MaxRequests int
	// 时间窗口大小
	Window time.Duration
	// 限流 key 前缀
	KeyPrefix string
	// 是否按用户限流（需要在 Authorization 之后使用）
	ByUser bool
	// 是否按 IP 限流
	ByIP bool
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: 100,          // 每分钟 100 次
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:",
		ByIP:        true,
		ByUser:      false,
	}
}

// RateLimit 创建限流中间件
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var key string

		if cfg.ByUser {
			uid := c.GetInt64("uid")
			if uid > 0 {
				key = fmt.Sprintf("%suser:%d", cfg.KeyPrefix, uid)
			}
		}

		if key == "" && cfg.ByIP {
			key = fmt.Sprintf("%sip:%s", cfg.KeyPrefix, c.ClientIP())
		}

		if key == "" {
			c.Next()
			return
		}

		// 使用 Redis 滑动窗口限流
		allowed, remaining, resetAt := checkRateLimit(key, cfg.MaxRequests, cfg.Window)

		// 设置响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkRateLimit 检查限流（滑动窗口算法）
// 返回: 是否允许, 剩余次数, 重置时间戳
func checkRateLimit(key string, maxRequests int, window time.Duration) (bool, int, int64) {
	ctx := context.Background()
	rds := dbs.Rds()
	now := time.Now()
	windowStart := now.Add(-window).UnixMilli()
	nowMs := now.UnixMilli()

	pipe := rds.Pipeline()

	// 移除窗口外的请求记录
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// 获取当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, key)

	// 添加当前请求
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(nowMs),
		Member: fmt.Sprintf("%d", nowMs),
	})

	// 设置过期时间
	pipe.Expire(ctx, key, window+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// Redis 错误时放行
		return true, maxRequests, now.Add(window).Unix()
	}

	count := int(countCmd.Val())
	remaining := maxRequests - count - 1
	if remaining < 0 {
		remaining = 0
	}
	resetAt := now.Add(window).Unix()

	if count >= maxRequests {
		return false, 0, resetAt
	}

	return true, remaining, resetAt
}

// StrictRateLimit 严格限流（用于敏感接口如登录）
func StrictRateLimit() gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		MaxRequests: 10,            // 每分钟 10 次
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:strict:",
		ByIP:        true,
	})
}

// APIRateLimit 普通 API 限流
func APIRateLimit() gin.HandlerFunc {
	return RateLimit(DefaultRateLimitConfig())
}
