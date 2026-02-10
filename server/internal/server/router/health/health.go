package health

import (
	"context"
	"server/pkg/dbs"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus 健康检查响应
type HealthStatus struct {
	Status    string            `json:"status"`    // ok, degraded, unhealthy
	Timestamp int64             `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// Routes 健康检查路由（无需认证）
func Routes(ge gin.IRouter) {
	ge.GET("health", healthCheck)
	ge.GET("health/live", liveCheck)   // 存活检查（用于 k8s liveness）
	ge.GET("health/ready", readyCheck) // 就绪检查（用于 k8s readiness）
}

// healthCheck 完整健康检查
func healthCheck(c *gin.Context) {
	status := HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
		Services:  make(map[string]string),
	}

	// 检查 MySQL
	if err := checkMySQL(); err != nil {
		status.Services["mysql"] = "unhealthy: " + err.Error()
		status.Status = "unhealthy"
	} else {
		status.Services["mysql"] = "ok"
	}

	// 检查 Redis
	if err := checkRedis(); err != nil {
		status.Services["redis"] = "unhealthy: " + err.Error()
		if status.Status == "ok" {
			status.Status = "degraded"
		}
	} else {
		status.Services["redis"] = "ok"
	}

	statusCode := 200
	if status.Status == "unhealthy" {
		statusCode = 503
	}

	c.JSON(statusCode, status)
}

// liveCheck 存活检查（只要服务在运行就返回 ok）
func liveCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// readyCheck 就绪检查（所有依赖服务正常才返回 ok）
func readyCheck(c *gin.Context) {
	mysqlErr := checkMySQL()
	redisErr := checkRedis()

	if mysqlErr != nil || redisErr != nil {
		c.JSON(503, gin.H{
			"status":    "not_ready",
			"timestamp": time.Now().Unix(),
			"mysql":     errToString(mysqlErr),
			"redis":     errToString(redisErr),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	})
}

func checkMySQL() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db := dbs.DBAdmin.DB()
	return db.PingContext(ctx)
}

func checkRedis() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return dbs.Rds().Ping(ctx).Err()
}

func errToString(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}
