package control

import (
	"context"
	"fmt"
	ctrl "server/internal/server/control"
	"server/pkg/result"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// listGostServices 列出 GOST 服务
func listGostServices(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "20")
	portStr := c.DefaultQuery("port", "0")
	serverIdStr := c.Query("server_id")

	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)
	port, _ := strconv.Atoi(portStr)

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		gostCtrl, _ := ctrl.GetGostCluster()
		data, err := gostCtrl.ListGostServicesOnServer(ctx, serverId, page, size, port)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.GetGost().ListGostServices(ctx, page, size, port)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// getGostService 获取单个 GOST 服务
func getGostService(c *gin.Context) {
	name := c.Param("name")
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		gostCtrl, _ := ctrl.GetGostCluster()
		data, err := gostCtrl.GetGostServiceOnServer(ctx, serverId, name)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.GetGost().GetGostService(ctx, name)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// createGostService 创建 GOST 服务
func createGostService(c *gin.Context) {
	var req struct {
		ServerId int                     `json:"server_id"`
		Config   *ctrl.GostServiceConfig `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && req.ServerId > 0 {
		gostCtrl, _ := ctrl.GetGostCluster()
		if err := gostCtrl.CreateGostServiceOnServer(ctx, req.ServerId, req.Config); err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, nil)
		return
	}

	if err := ctrl.GetGost().CreateGostService(ctx, req.Config); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// updateGostService 更新 GOST 服务
func updateGostService(c *gin.Context) {
	name := c.Param("name")
	var req struct {
		ServerId int                     `json:"server_id"`
		Config   *ctrl.GostServiceConfig `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && req.ServerId > 0 {
		gostCtrl, _ := ctrl.GetGostCluster()
		if err := gostCtrl.UpdateGostServiceOnServer(ctx, req.ServerId, name, req.Config); err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, nil)
		return
	}

	if err := ctrl.GetGost().UpdateGostService(ctx, name, req.Config); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// deleteGostService 删除 GOST 服务
func deleteGostService(c *gin.Context) {
	name := c.Param("name")
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		gostCtrl, _ := ctrl.GetGostCluster()
		if err := gostCtrl.DeleteGostServiceOnServer(ctx, serverId, name); err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, nil)
		return
	}

	if err := ctrl.GetGost().DeleteGostService(ctx, name); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// listGostChains 列出 GOST 链
func listGostChains(c *gin.Context) {
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		gostCtrl, _ := ctrl.GetGostCluster()
		data, err := gostCtrl.ListGostChainsOnServer(ctx, serverId)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.GetGost().ListGostChains(ctx)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// persistGostConfig 持久化 GOST 配置
func persistGostConfig(c *gin.Context) {
	var req struct {
		ServerId int `json:"server_id"`
	}
	c.ShouldBindJSON(&req)

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && req.ServerId > 0 {
		gostCtrl, _ := ctrl.GetGostCluster()
		if err := gostCtrl.PersistGostConfigOnServer(ctx, req.ServerId); err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, nil)
		return
	}

	if err := ctrl.GetGost().PersistGostConfig(ctx); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// getGostConfigSyncStatus 获取 GOST 配置同步状态
func getGostConfigSyncStatus(c *gin.Context) {
	serverIdStr := c.Query("server_id")

	ctx, cancel := newCtx()
	defer cancel()

	if ctrl.IsClusterMode() && serverIdStr != "" {
		serverId, err := strconv.Atoi(serverIdStr)
		if err != nil {
			result.GParamErr(c, err)
			return
		}
		gostCtrl, _ := ctrl.GetGostCluster()
		data, err := gostCtrl.GetGostConfigSyncStatusOnServer(ctx, serverId)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, data)
		return
	}

	data, err := ctrl.GetGost().GetGostConfigSyncStatus(ctx)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// tunnelCheck 隧道检测
func tunnelCheck(c *gin.Context) {
	var req ctrl.TunnelCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	data, err := ctrl.GetTunnel().TunnelCheck(ctx, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// gostOneClickDeploy GOST 一键部署（流式 SSE 响应）
// 单机模式：零参数，全自动安装+配置
// 多机模式：需要 server_id + merchant_ids
func gostOneClickDeploy(c *gin.Context) {
	var req ctrl.GostDeployRequest
	c.ShouldBindJSON(&req)

	// 流式响应
	result.GStream(c)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := ctrl.GostOneClickDeploy(ctx, req, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(c, false, err.Error())
		return
	}
	result.GStreamEnd(c, true, "部署完成")
}
