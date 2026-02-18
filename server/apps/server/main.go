package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"server/internal/server"
	"server/internal/server/cfg"
	"syscall"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	Version   = "dev"
	BuildTime = ""
	GoVersion = ""
)

var (
	configFile  = flag.String("f", "config.yaml", "the config file")
	showVersion = flag.Bool("version", false, "show version info")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("版本: %s\n构建时间: %s\nGo版本: %s\n正式: %s\n", Version, BuildTime, GoVersion, cfg.Release)
		return
	}

	if err := conf.Load(*configFile, &cfg.C); err != nil {
		logx.Errorf("Load config error: %v", err)
		os.Exit(1)
	}

	// 从环境变量覆盖配置
	cfg.ApplyEnvOverrides()

	// 配置日志
	cfg.C.MustSetUp()

	logx.Info("===========================================")
	logx.Infof("  Control Server Starting... (v%s)", Version)
	logx.Info("===========================================")

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go server.Serve(ctx)

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logx.Info("Received shutdown signal")
	cancelFunc()
}
