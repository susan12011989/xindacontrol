package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"server/internal/buildworker"
	"server/internal/buildworker/cfg"
	"syscall"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "config.yaml", "the config file")

func main() {
	flag.Parse()

	if err := conf.Load(*configFile, &cfg.C); err != nil {
		logx.Errorf("Load config error: %v", err)
		os.Exit(1)
	}

	// 配置日志
	cfg.C.MustSetUp()

	logx.Info("===========================================")
	logx.Info("  Build Worker Service Starting...")
	logx.Info("===========================================")

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go buildworker.Serve(ctx)

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logx.Info("Received shutdown signal")
	cancelFunc()
}
