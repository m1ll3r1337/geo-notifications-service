package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/m1ll3r1337/geo-notifications-service/internal/app"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/config"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	logLevel := logger.ParseLogLevel(cfg.Log.Level)
	log := logger.New(os.Stdout, logLevel, "GEO")

	ctx := context.Background()
	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))
	log.BuildInfo(ctx)

	srvCfg := app.Config{
		LogLevel: cfg.Log.Level,
		Addr:     cfg.HTTP.Addr,
	}
	s := app.NewServer(srvCfg, logger.NewStdLogger(log, logger.LevelError))

	log.Info(ctx, "starting server", "address", cfg.HTTP.Addr)
	if err := s.Start(); err != nil {
		log.Error(ctx, "server failed to start", "error", err)
		return
	}
}
