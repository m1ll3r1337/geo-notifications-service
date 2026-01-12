package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Config struct {
	URL             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

func Open(ctx context.Context, cfg Config) (*sqlx.DB, error) {
	if cfg.URL == "" {
		return nil, errors.New("db url is required")
	}

	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = 5 * time.Second
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = time.Hour
	}

	db, err := sqlx.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}

func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	for attempts := 1; ; attempts++ {
		if err := db.PingContext(ctx); err == nil {
			break
		}

		sleep := time.Duration(attempts) * 100 * time.Millisecond
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	const q = `SELECT TRUE`
	var tmp bool
	if err := db.QueryRowContext(ctx, q).Scan(&tmp); err != nil {
		return fmt.Errorf("db status query: %w", err)
	}
	return nil
}
