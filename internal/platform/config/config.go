// Package config handles environment-based configuration loading.
package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	HTTP struct {
		Addr string `default:":8080"`
	}
	Log struct {
		Level string `default:"info"`
	}
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("GEO", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
