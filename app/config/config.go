package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	DataDir string

	StockPath     string
	PortfolioPath string

	ServerPort int    `env:"API_INT_PORT,default=8000"`
	ServerHOST string `env:"SERVER_HOST,default=0.0.0.0"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	cfg := &Config{DataDir: "/data"}
	if err := envconfig.Process(ctx, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	info, err := os.Stat(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("data dir %q: %w", cfg.DataDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("data dir %q is not a directory", cfg.DataDir)
	}

	cfg.StockPath, err = dataFilePath(cfg.DataDir, "stocks.yml")
	if err != nil {
		return nil, err
	}
	cfg.PortfolioPath, err = dataFilePath(cfg.DataDir, "portfolio.yml")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func dataFilePath(dataDir, name string) (string, error) {
	path := filepath.Join(dataDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, expected a file", path)
	}
	return path, nil
}
