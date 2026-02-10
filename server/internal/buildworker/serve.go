package buildworker

import (
	"context"
	"server/internal/buildworker/artifact"
	"server/internal/buildworker/cfg"
	"server/internal/buildworker/queue"
	"server/internal/buildworker/worker"
	"server/pkg/dbs"

	"github.com/zeromicro/go-zero/core/logx"
)

func Serve(ctx context.Context) {
	// 初始化数据库和 Redis
	dbs.InitMysql(cfg.C.Mysql, &dbs.DBAdmin)
	dbs.InitRedis(cfg.C.Redis)

	logx.Info("Database and Redis initialized")

	// 创建组件
	buildQueue := queue.NewBuildQueue(dbs.Rds())
	uploader := artifact.NewUploader()
	buildWorker := worker.NewWorker(buildQueue, uploader)

	logx.Info("Build worker components created")

	// 启动 Worker
	go buildWorker.Start()

	logx.Info("Build worker service started")
	logx.Infof("  Max concurrent: %d", cfg.C.Worker.MaxConcurrent)
	logx.Infof("  Poll interval: %d seconds", cfg.C.Worker.PollInterval)
	logx.Infof("  Build timeout: %d seconds", cfg.C.Worker.BuildTimeout)
	logx.Infof("  Scripts dir: %s", cfg.C.Worker.ScriptsDir)
	logx.Infof("  Output dir: %s", cfg.C.Worker.OutputDir)
	logx.Infof("  Storage type: %s", cfg.C.Storage.Type)

	// 等待退出信号
	<-ctx.Done()

	// 优雅停止
	logx.Info("Shutting down build worker...")
	buildWorker.Stop()
	logx.Info("Build worker service stopped")
}
