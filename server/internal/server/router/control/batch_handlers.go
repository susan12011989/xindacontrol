package control

import (
	"context"
	ctrl "server/internal/server/control"
	"server/internal/server/middleware"
	"server/pkg/result"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterBatchRoutes 注册批量运维路由
func RegisterBatchRoutes(group gin.IRouter) {
	batch := group.Group("batch")
	batch.POST("service-action", batchServiceAction)
	batch.POST("health-check", batchHealthCheck)
}

// batchServiceAction 批量服务操作
func batchServiceAction(c *gin.Context) {
	var req struct {
		ServerIds   []int  `json:"server_ids" binding:"required"`
		ServiceName string `json:"service_name" binding:"required"`
		Action      string `json:"action" binding:"required,oneof=start stop restart"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if !ctrl.IsClusterMode() {
		// 单机模式直接操作本地
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		data, err := ctrl.Get().ServiceAction(ctx, ctrl.ServiceName(req.ServiceName), ctrl.ServiceAction(req.Action))
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"results": []any{data}})
		return
	}

	cluster, _ := ctrl.GetCluster()
	operator := middleware.GetUsername(c)
	if operator == "" {
		operator = "admin"
	}

	type actionResult struct {
		ServerId int         `json:"server_id"`
		Result   interface{} `json:"result"`
		Error    string      `json:"error,omitempty"`
	}

	var results []actionResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sid := range req.ServerIds {
		serverId := sid
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			data, err := cluster.ServiceActionOnServer(ctx, serverId, ctrl.ServiceName(req.ServiceName), ctrl.ServiceAction(req.Action), operator)

			mu.Lock()
			r := actionResult{ServerId: serverId, Result: data}
			if err != nil {
				r.Error = err.Error()
			}
			results = append(results, r)
			mu.Unlock()
		}()
	}
	wg.Wait()

	result.GOK(c, gin.H{"results": results})
}

// batchHealthCheck 批量健康检查
func batchHealthCheck(c *gin.Context) {
	var req struct {
		ServerIds []int `json:"server_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if !ctrl.IsClusterMode() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		data, err := ctrl.Get().HealthCheck(ctx)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"results": []any{data}})
		return
	}

	cluster, _ := ctrl.GetCluster()

	type healthResult struct {
		ServerId int         `json:"server_id"`
		Health   interface{} `json:"health"`
		Error    string      `json:"error,omitempty"`
	}

	var results []healthResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sid := range req.ServerIds {
		serverId := sid
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			data, err := cluster.HealthCheckOnServer(ctx, serverId)

			mu.Lock()
			r := healthResult{ServerId: serverId, Health: data}
			if err != nil {
				r.Error = err.Error()
			}
			results = append(results, r)
			mu.Unlock()
		}()
	}
	wg.Wait()

	result.GOK(c, gin.H{"results": results})
}
