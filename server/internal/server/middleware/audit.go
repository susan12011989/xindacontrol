package middleware

import (
	"bytes"
	"io"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

// AuditControl 控制层操作审计中间件
// 记录所有 POST/PUT/DELETE 操作到审计日志表
func AuditControl(c *gin.Context) {
	// 只审计写操作
	method := c.Request.Method
	if method != "POST" && method != "PUT" && method != "DELETE" {
		c.Next()
		return
	}

	// 读取请求体（需要重新写回供后续 handler 读取）
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// 记录开始时间
	start := time.Now()

	// 执行后续 handler
	c.Next()

	// 异步写入审计日志（不阻塞响应）
	go func() {
		username := GetUsername(c)
		if username == "" {
			username = "unknown"
		}

		path := c.Request.URL.Path
		action := deriveAction(method, path)
		targetType, targetName := deriveTarget(path)

		// 限制 body 大小（避免大文件上传写入审计）
		detail := string(bodyBytes)
		if len(detail) > 2048 {
			detail = detail[:2048] + "...(truncated)"
		}

		status := "success"
		errorMsg := ""
		if c.Writer.Status() >= 400 {
			status = "failed"
			errorMsg = c.Errors.String()
		}

		log := entity.AuditLogs{
			Username:   username,
			Action:     action,
			TargetType: targetType,
			TargetName: targetName,
			Detail:     detail,
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Status:     status,
			ErrorMsg:   errorMsg,
			CreatedAt:  start,
		}

		if _, err := dbs.DBAdmin.Insert(&log); err != nil {
			logx.Errorf("[Audit] 写入审计日志失败: %v", err)
		}
	}()
}

// deriveAction 从请求方法和路径推导操作类型
func deriveAction(method, path string) string {
	// control/service/action → service_action
	// control/gost/one-click-deploy → gost_deploy
	parts := strings.Split(strings.TrimPrefix(path, "/server/v1/control/"), "/")
	if len(parts) >= 2 {
		return strings.ReplaceAll(strings.Join(parts[:2], "_"), "-", "_")
	}
	if len(parts) == 1 {
		return method + "_" + strings.ReplaceAll(parts[0], "-", "_")
	}
	return method + "_unknown"
}

// deriveTarget 从路径推导目标类型和名称
func deriveTarget(path string) (targetType, targetName string) {
	if strings.Contains(path, "gost") {
		return "gost", ""
	}
	if strings.Contains(path, "service") {
		return "service", ""
	}
	if strings.Contains(path, "tunnel") {
		return "tunnel", ""
	}
	if strings.Contains(path, "batch") {
		return "batch", ""
	}
	if strings.Contains(path, "config") {
		return "config", ""
	}
	if strings.Contains(path, "deploy") {
		return "deploy", ""
	}
	return "control", ""
}
