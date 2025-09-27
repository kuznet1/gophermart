package config

import (
	"errors"
	"flag"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	MigrationsPath       string `env:"MIGRATIONS_PATH"`
	SecretKey            string `env:"SECRET_KEY"`
}

func NewConfig() (Config, error) {
	cfg := Config{}
	flag.StringVar(&cfg.RunAddress, "a", ":8086", "Server address")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "localhost:8080", "Accrual service address")
	flag.StringVar(&cfg.DatabaseURI, "d", "postgres://postgres@localhost:5432/gophermart", "Database URI")
	flag.StringVar(&cfg.MigrationsPath, "m", "file://migrations", "Migrations path")
	flag.StringVar(&cfg.SecretKey, "k", "", "secret key for cookie signing")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		return Config{}, err
	}
	if cfg.SecretKey == "" {
		return Config{}, errors.New("secret key is empty")
	}
	return cfg, err

}
