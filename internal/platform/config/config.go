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
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("GEO", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
