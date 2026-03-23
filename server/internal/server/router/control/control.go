package control

import (
	"context"
	"net/http"
	ctrl "server/internal/server/control"
	"server/internal/server/middleware"
	"server/pkg/result"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Routes 统一控制层路由
// 同时兼容单机和多机模式，根据配置自动切换
func Routes(gi gin.IRouter) {
	group := gi.Group("control")
	group.Use(middleware.Authorization)
	group.Use(middleware.AuditControl) // 操作审计日志

	// 元信息
	group.GET("mode", getMode)
	group.GET("health", healthCheck)

	// 服务生命周期
	group.POST("service/action", serviceAction)
	group.GET("service/status", getServiceStatus)
	group.GET("service/logs", getServiceLogs)

	// 配置管理
	group.GET("config", getConfigFile)
	group.POST("config", updateConfigFile)

	// 文件部署
	group.POST("deploy", deployBinary)

	// 监控
	group.GET("stats", getServerStats)
	group.GET("docker/containers", getDockerContainers)

	// 服务发现
	group.GET("endpoints", getEndpoints)

	// GOST 管理
	group.GET("gost/services", listGostServices)
	group.GET("gost/services/:name", getGostService)
	group.POST("gost/services", createGostService)
	group.PUT("gost/services/:name", updateGostService)
	group.DELETE("gost/services/:name", deleteGostService)
	group.GET("gost/chains", listGostChains)
	group.POST("gost/config/persist", persistGostConfig)
	group.GET("gost/config/sync-status", getGostConfigSyncStatus)

	// GOST 一键部署
	group.POST("gost/one-click-deploy", gostOneClickDeploy)

	// 隧道检测
	group.POST("tunnel/check", tunnelCheck)

	// 批量运维
	RegisterBatchRoutes(group)
}

func newCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// getMode 获取当前控制模式
func getMode(c *gin.Context) {
	controller := ctrl.Get()
	result.GOK(c, gin.H{
		"mode": controller.Mode(),
	})
}

// healthCheck 健康检查
func healthCheck(c *gin.Context) {
	ctx, cancel := newCtx()
	defer cancel()

	controller := ctrl.Get()

	// 多机模式支持指定 server_id
	if ctrl.IsClusterMode() {
		serverIdStr := c.Query("server_id")
		if serverIdStr != "" {
			serverId, err := strconv.Atoi(serverIdStr)
			if err != nil {
				result.GParamErr(c, err)
				return
			}
			cluster, _ := ctrl.GetCluster()
			data, err := cluster.HealthCheckOnServer(ctx, serverId)
			if err != nil {
				result.GErr(c, err)
				return
			}
			result.GOK(c, data)
			return
		}
	}

	data, err := controller.HealthCheck(ctx)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// serviceAction 服务操作（start/stop/restart）
func serviceAction(c *gin.Context) {
	var req struct {
		ServerId    int    `json:"server_id"`                                             // 多机模式需要
		ServiceName string `json:"service_name" binding:"required"`                       // server/wukongim/gost
		Action      string `json:"action" binding:"required,oneof=start stop restart"`    // 操作
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	svc := ctrl.ServiceName(req.ServiceName)
	action := ctrl.ServiceAction(req.Action)

	if ctrl.IsClusterMode() && req.ServerId > 0 {
		cluster, _ := ctrl.GetCluster()
		operator := middleware.GetUsername(c)
		if operator == "" {
			operator = "admin"
		}
		data, err := cluster.ServiceActionOnServer(ctx, req.ServerId, svc, action, operator)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.Get().ServiceAction(ctx, svc, action)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// getServiceStatus 获取服务状态
func getServiceStatus(c *gin.Context) {
	serviceName := c.Query("service_name")
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	svc := ctrl.ServiceName(serviceName)

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		cluster, _ := ctrl.GetCluster()
		data, err := cluster.GetServiceStatusOnServer(ctx, serverId, svc)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"services": data})
		return
	}

	data, err := ctrl.Get().GetServiceStatus(ctx, svc)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"services": data})
}

// getServiceLogs 获取服务日志
func getServiceLogs(c *gin.Context) {
	serviceName := c.Query("service_name")
	serverIdStr := c.Query("server_id")
	linesStr := c.DefaultQuery("lines", "100")
	lines, _ := strconv.Atoi(linesStr)

	if serviceName == "" {
		result.GParamErr(c, nil)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	svc := ctrl.ServiceName(serviceName)

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		cluster, _ := ctrl.GetCluster()
		data, err := cluster.GetServiceLogsOnServer(ctx, serverId, svc, lines)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.Get().GetServiceLogs(ctx, svc, lines)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// getConfigFile 获取服务配置文件
func getConfigFile(c *gin.Context) {
	serviceName := c.Query("service_name")
	serverIdStr := c.Query("server_id")

	if serviceName == "" {
		result.GParamErr(c, nil)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	svc := ctrl.ServiceName(serviceName)

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		cluster, _ := ctrl.GetCluster()
		data, err := cluster.GetConfigFileOnServer(ctx, serverId, svc)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.Get().GetConfigFile(ctx, svc)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// updateConfigFile 更新服务配置文件
func updateConfigFile(c *gin.Context) {
	var req struct {
		ServerId    int    `json:"server_id"`
		ServiceName string `json:"service_name" binding:"required"`
		Content     string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	svc := ctrl.ServiceName(req.ServiceName)

	if ctrl.IsClusterMode() && req.ServerId > 0 {
		cluster, _ := ctrl.GetCluster()
		data, err := cluster.UpdateConfigFileOnServer(ctx, req.ServerId, svc, req.Content)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.Get().UpdateConfigFile(ctx, svc, req.Content)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// deployBinary 部署服务二进制
func deployBinary(c *gin.Context) {
	serviceName := c.PostForm("service_name")
	serverIdStr := c.PostForm("server_id")

	if serviceName == "" {
		result.GParamErr(c, nil)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择文件"})
		return
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	svc := ctrl.ServiceName(serviceName)

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		cluster, _ := ctrl.GetCluster()
		path, err := cluster.DeployBinaryToServer(ctx, serverId, svc, header.Filename, file)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"path": path})
		return
	}

	path, err := ctrl.Get().DeployBinary(ctx, svc, header.Filename, file)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"path": path})
}

// getServerStats 获取服务器资源
func getServerStats(c *gin.Context) {
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		cluster, _ := ctrl.GetCluster()
		data, err := cluster.GetServerStatsOnServer(ctx, serverId)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.Get().GetServerStats(ctx)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// getDockerContainers 获取 Docker 容器列表
func getDockerContainers(c *gin.Context) {
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		cluster, _ := ctrl.GetCluster()
		data, err := cluster.GetDockerContainersOnServer(ctx, serverId)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"containers": data})
		return
	}

	data, err := ctrl.Get().GetDockerContainers(ctx)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"containers": data})
}

// getEndpoints 获取服务端点
func getEndpoints(c *gin.Context) {
	serviceName := c.Query("service_name")
	if serviceName == "" {
		result.GParamErr(c, nil)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	svc := ctrl.ServiceName(serviceName)
	data, err := ctrl.Get().GetEndpoints(ctx, svc)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"endpoints": data})
}
