package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/m1ll3r1337/geo-notifications-service/internal/app/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/http"
	"github.com/m1ll3r1337/geo-notifications-service/internal/http/handlers"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/config"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db"
	healthdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/health"
	incidentsdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/txrunner"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/uow"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
	incidentscache "github.com/m1ll3r1337/geo-notifications-service/internal/platform/redis/cache"
	healthredis "github.com/m1ll3r1337/geo-notifications-service/internal/platform/redis/health"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/redis/queue"
	"github.com/m1ll3r1337/geo-notifications-service/internal/workers/outboxrelay"
	webhookworker "github.com/m1ll3r1337/geo-notifications-service/internal/workers/webhook"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
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

	// --- Redis Cache ---
	cacheRdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.CacheDB,
	})
	defer cacheRdb.Close()

	if err := cacheRdb.Ping(ctx).Err(); err != nil {
		log.Error(ctx, "startup", "status", "redis cache init failed", "error", err)
		return
	}

	// --- Redis Queue ---
	queueRdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.QueueDB,
	})
	defer queueRdb.Close()

	if err := queueRdb.Ping(ctx).Err(); err != nil {
		log.Error(ctx, "startup", "status", "redis queue init failed", "error", err)
		return
	}

	// --- Incidents module wiring ---
	baseRepo := incidentsdb.New(sqlDB)

	cachedRepo := incidentscache.New(
		cacheRdb,
		baseRepo,
		incidentscache.WithTTL(time.Duration(cfg.Cache.ActiveIncidentsTTLSeconds)*time.Second),
		incidentscache.WithLogger(log),
	)
	incEow := uow.New(sqlDB)
	incTxRunner := txrunner.NewIncidentsTxRunner(incEow)
	incSvc := incidents.NewService(cachedRepo, incTxRunner)
	incHandlers := handlers.NewIncidents(incSvc, time.Duration(cfg.Stats.TimeWindowMinutes)*time.Minute)

	// --- System ---
	sysHandler := handlers.NewSystem(
		log,
		handlers.Dependency{
			Name:   "postgres",
			Pinger: healthdb.NewPostgresPinger(sqlDB),
		},
		handlers.Dependency{
			Name:   "redis_queue",
			Pinger: healthredis.NewRedisPinger(queueRdb),
		},
		handlers.Dependency{
			Name:   "redis_cache",
			Pinger: healthredis.NewRedisPinger(cacheRdb),
		},
	)

	// --- HTTP ---
	router := http.NewRouter(log, logLevel, incHandlers, sysHandler, cfg.Security.ApiKey)
	s := http.NewServer(http.Config{Addr: cfg.HTTP.Addr}, router, logger.NewStdLogger(log, logger.LevelError))

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		log.Info(ctx, "startup", "status", "server started", "addr", cfg.HTTP.Addr)
		serverErrors <- s.Start()
	}()

	// --- Workers ---
	workerCtx, workerCancel := context.WithCancel(ctx)

	outboxRelay := outboxrelay.New(sqlDB, queue.New(queueRdb, cfg.Workers.OutboxRelay.Stream), log)
	webhookWorker := webhookworker.New(
		queueRdb,
		cfg.Workers.Webhook.Stream,
		cfg.Workers.Webhook.Group,
		cfg.Workers.Webhook.Consumer,
		cfg.Workers.Webhook.URL,
		log,
	)

	g, gctx := errgroup.WithContext(workerCtx)

	g.Go(func() error { return outboxRelay.Run(gctx) })
	g.Go(func() error { return webhookWorker.Run(gctx) })

	select {
	case err := <-serverErrors:
		log.Error(ctx, "startup", "status", "server failed", "error", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		workerCancel()

		if err := s.Shutdown(shutdownCtx); err != nil {
			log.Error(ctx, "could not stop server gracefully", "error", err)
			_ = s.Close()
		}
	}

	if err := g.Wait(); err != nil {
		log.Error(ctx, "shutdown", "status", "workers failed to stop gracefully", "error", err)
	}
}
