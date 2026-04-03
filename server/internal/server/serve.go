package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"server/internal/server/cfg"
	ctrl "server/internal/server/control"
	"server/internal/server/cron"
	"server/internal/server/middleware"
	"server/internal/server/router/alert"
	"server/internal/server/router/announcements"
	"server/internal/server/router/audit"
	"server/internal/server/router/auth"
	"server/internal/server/router/clients"
	"server/internal/server/router/cloud_account"
	"server/internal/server/router/cloud_aliyun"
	"server/internal/server/router/cloud_aws"
	"server/internal/server/router/cloud_tencent"
	controlRouter "server/internal/server/router/control"
	"server/internal/server/router/deploy"
	deployService "server/internal/server/service/deploy"
	"server/internal/server/router/docker"
	"server/internal/server/router/wukongim"
	"server/internal/server/router/feature"
	"server/internal/server/router/global"
	"server/internal/server/router/health"
	"server/internal/server/router/ip_embed"
	"server/internal/server/router/merchant"
	"server/internal/server/router/merchant_api"
	"server/internal/server/router/merchant_storage"
	"server/internal/server/router/project"
	"server/internal/server/router/resource_group"
	"server/internal/server/router/resource_overview"
	"server/internal/server/router/utils"
	"server/internal/server/static"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"server/pkg/result"
	"server/pkg/token_manager"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

func Serve(ctx context.Context) {
	dbs.InitMysql(cfg.C.Mysql, &dbs.DBAdmin)
	dbs.InitRedis(cfg.C.Redis)
	token_manager.Init()
	gostapi.InitTaskQueue(dbs.Rds()) // 初始化 GOST 任务队列

	// 初始化统一控制层（根据配置切换单机/多机模式）
	controlMode := ctrl.ModeCluster // 默认多机模式
	if cfg.C.Control != nil && cfg.C.Control.Mode != "" {
		controlMode = ctrl.Mode(cfg.C.Control.Mode)
	}
	ctrl.Init(ctrl.Config{Mode: controlMode})

	// 启动 GOST 链路健康监控（后台定时检测，异常记入告警日志）
	gostMonitor := cron.NewGostMonitor(2 * time.Minute)
	gostMonitor.Start()
	defer gostMonitor.Stop()

	// 自动同步所有数据库表（不存在则创建，字段变更则自动加列）
	autoSyncTables := []interface{}{
		new(entity.AdminUsers),
		new(entity.Merchants),
		new(entity.Servers),
		new(entity.CloudAccounts),
		new(entity.TlsCertificates),
		new(entity.DeployConfigs),
		new(entity.DeployHistory),
		new(entity.DockerOperationHistory),
		new(entity.AnnouncementLogs),
		new(entity.TunnelStatsRecord),
		new(entity.GlobalOssUrl),
		new(entity.Clients),
		new(entity.IpEmbedSelections),
		new(entity.IpEmbedTargets),
		new(entity.FeatureFlags),
		new(entity.MerchantOssConfigs),
		new(entity.MerchantGostServers),
		new(entity.AuditLogs),
		new(entity.AlertRules),
		new(entity.AlertLogs),
		new(entity.Projects),
		new(entity.ProjectGostServers),
		new(entity.MerchantStorageConfigs),
		new(entity.ResourceTags),
		new(entity.ResourceTagRelations),
		new(entity.ResourceGroups),
		new(entity.CloudInstanceBindings),
	}
	for _, table := range autoSyncTables {
		if err := dbs.DBAdmin.Sync2(table); err != nil {
			logx.Errorf("Sync table failed: %v", err)
		}
	}
	// http api
	ge := gin.Default()

	// 禁用自动重定向（避免301问题）
	ge.RedirectTrailingSlash = false
	ge.RedirectFixedPath = false

	// CORS 配置
	ge.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},                                                               // 允许所有来源
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},                         // 允许的 HTTP 方法
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},        // 允许的请求头
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-RateLimit-Remaining"},        // 暴露的响应头
		AllowCredentials: true,                                                                        // 允许携带认证信息
		MaxAge:           12 * time.Hour,                                                              // 预检请求结果缓存时间
	}))

	// 健康检查路由（无需认证，放在最前面）
	health.Routes(ge)

	group := ge.Group("/server/v1")
	group.Use(middleware.RequestID())        // 请求追踪
	group.Use(middleware.APIRateLimit())     // API限流
	// group.Use(middleware.IPWhiteList)     // IP白名单
	group.Use(middleware.LogRequest)         // 请求日志
	group.Use(middleware.NoDestructiveOps)   // maintainer 角色禁止删除/销毁操作
	group.GET("ping", func(c *gin.Context) { // 测试是否能通
		result.GOK(c, map[string]any{
			"timestamp": time.Now().Unix(),
		})
	})
	auth.Routes(group)          // 登录认证 2FA管理
	merchant.Routes(group)      // 商户管理
	project.Routes(group)       // 项目管理
	merchant_api.Routes(group)  // 商户API（供商户服务调用，Basic Auth认证）
	cloud_account.Routes(group) // 系统云账号管理
	cloud_aliyun.Routes(group)  // 阿里云管理
	cloud_aws.Routes(group)     // AWS 管理
	cloud_tencent.Routes(group) // 腾讯云管理
	deploy.Routes(group)        // 部署管理（服务器、服务进程）
	docker.Routes(group)        // Docker容器管理
	feature.Routes(group)       // 功能开关管理
	utils.Routes(group)         // 工具类接口（端口转换、IP工具）
	ip_embed.Routes(group)      // IP嵌入上传
	global.Routes(group)        // 全局管理
	announcements.Routes(group) // 系统公告
	audit.Routes(group)            // 操作审计日志
	alert.Routes(group)            // 告警通知管理
	merchant_storage.Routes(group)  // 商户存储配置管理
	resource_group.Routes(group)    // 资源分组管理
	resource_overview.Routes(group) // 资源总览
	clients.Routes(group)           // 客户端管理
	controlRouter.Routes(group)     // 统一控制层（兼容单机/多机）
	wukongim.Routes(group)          // WuKongIM监控管理

	// 配置静态文件服务（前端页面 + SPA 回退）
	fsys := static.FS()
	spa := static.NewSPAHandler(fsys, "index.html")

	// 前端 publicPath 前缀（对应 VITE_PUBLIC_PATH）
	const spaPrefix = "/hxdadmin"

	// 上传资源的静态文件服务（Logo等）- 支持环境变量 ASSETS_DIR 覆盖
	ge.Static("/assets", consts.AssetsDir)

	// 非 API 的其它路径走 SPA 回退；API 仍返回 JSON 404
	ge.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/server/v1") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API接口不存在"})
			return
		}
		// 根路径重定向到前端 publicPath
		if c.Request.URL.Path == "/" {
			c.Redirect(http.StatusFound, spaPrefix+"/")
			return
		}
		// 剥离前端 publicPath 前缀后查找嵌入的静态资源
		if strings.HasPrefix(c.Request.URL.Path, spaPrefix) {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, spaPrefix)
			if c.Request.URL.Path == "" {
				c.Request.URL.Path = "/"
			}
		}
		spa.ServeHTTP(c.Writer, c.Request)
	})

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         cfg.C.ListenOn,
		Handler:      ge,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器（非阻塞）
	go func() {
		logx.Infof("Started admin server on %s", cfg.C.ListenOn)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.Errorf("Server error: %v", err)
		}
	}()

	// 启动隧道连接数定期采集（每5分钟）
	tunnelTicker := time.NewTicker(5 * time.Minute)
	go func() {
		// 启动后先采集一次
		deployService.CollectAndSaveTunnelStats()
		for range tunnelTicker.C {
			deployService.CollectAndSaveTunnelStats()
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		logx.Info("Context cancelled, shutting down...")
	case sig := <-quit:
		logx.Infof("Received signal %v, shutting down...", sig)
	}

	// 优雅关闭，等待最多 30 秒
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logx.Errorf("Server forced to shutdown: %v", err)
	}

	logx.Info("Server exited gracefully")
}
