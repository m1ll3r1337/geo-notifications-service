package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

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
		Addr: cfg.HTTP.Addr,
	}

	// -------------------------------------------------------------------------
	// Start API Service
	router := app.NewRouter(log, logLevel)
	s := app.NewServer(srvCfg, router, logger.NewStdLogger(log, logger.LevelError))

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		log.Info(ctx, "startup", "status", "api router started", "addr", cfg.HTTP.Addr)

		serverErrors <- s.Start()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <-serverErrors:
		log.Error(ctx, "startup", "status", "server failed to start", "error", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.Shutdown(shutdownCtx); err != nil {
			log.Error(ctx, "could not stop server gracefully", "error", err)
			_ = s.Close()
		}
	}

}
