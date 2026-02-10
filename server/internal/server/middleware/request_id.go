package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader 请求ID的HTTP头名称
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey 在 gin.Context 中存储请求ID的 key
	RequestIDKey = "request_id"
)

// RequestID 请求追踪中间件
// 为每个请求分配唯一ID，方便日志追踪和问题排查
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用客户端传入的请求ID（用于分布式追踪）
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 存储到 context 中
		c.Set(RequestIDKey, requestID)

		// 设置响应头
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID 从 context 获取请求ID
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		return id.(string)
	}
	return ""
}