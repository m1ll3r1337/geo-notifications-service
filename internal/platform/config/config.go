// Package config handles environment-based configuration loading.
package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP struct {
		Addr string `default:":8080"`
	}
	Log struct {
		Level string `default:"info"`
	}
	DB struct {
		URL             string        `required:"true"`
		MaxIdleConns    int           `default:"2"`
		MaxOpenConns    int           `default:"10"`
		ConnMaxLifetime time.Duration `default:"1h"`
		ConnMaxIdleTime time.Duration `default:"0"`
		PingTimeout     time.Duration `default:"5s"`
	}
	Redis struct {
		Addr     string        `default:"localhost:6379"`
		Password string        `default:""`
		DB       int           `default:"0"`
		Timeout  time.Duration `default:"5s"`
	}
	Workers struct {
		Webhook struct {
			Stream   string `default:"webhook_events"`
			Group    string `default:"webhook_group"`
			Consumer string `default:"webhook_consumer"`
			URL      string `default:"http://localhost:3000/webhook"`
		}
		OutboxRelay struct {
			Stream string `default:"webhook_events"`
		}
	}
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("GEO", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
