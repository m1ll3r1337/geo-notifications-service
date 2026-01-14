package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/locations"
	"github.com/m1ll3r1337/geo-notifications-service/internal/http"
	"github.com/m1ll3r1337/geo-notifications-service/internal/http/handlers"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/config"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db"
	incidentsdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/incidents"
	locationsdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/locations"
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

	// --- DB ---
	sqlDB, err := db.Open(ctx, db.Config{
		URL:             cfg.DB.URL,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.DB.ConnMaxIdleTime,
		PingTimeout:     cfg.DB.PingTimeout,
	})
	if err != nil {
		log.Error(ctx, "startup", "status", "db init failed", "error", err)
		return
	}
	defer sqlDB.Close()

	if err := db.StatusCheck(ctx, sqlDB); err != nil {
		log.Error(ctx, "startup", "status", "db not ready", "error", err)
		return
	}

	// --- Incidents module wiring ---
	incRepo := incidentsdb.New(sqlDB)
	incSvc := incidents.NewService(incRepo)
	incHandlers := handlers.NewIncidents(incSvc)

	// --- Location module wiring ---
	locRepo := locationsdb.New(sqlDB)
	locSvc := locations.NewService(locRepo)
	locHandlers := handlers.NewLocations(locSvc)

	// --- HTTP ---
	router := http.NewRouter(log, logLevel, incHandlers, locHandlers)
	s := http.NewServer(http.Config{Addr: cfg.HTTP.Addr}, router, logger.NewStdLogger(log, logger.LevelError))

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		log.Info(ctx, "startup", "status", "server started", "addr", cfg.HTTP.Addr)
		serverErrors <- s.Start()
	}()

	select {
	case err := <-serverErrors:
		log.Error(ctx, "startup", "status", "server failed", "error", err)

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
