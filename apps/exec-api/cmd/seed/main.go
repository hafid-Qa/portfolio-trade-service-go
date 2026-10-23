package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"app/config"
	"app/internal/repositories/postgres"
	"app/internal/sqlc"

	"github.com/goccy/go-yaml"
)

type stockDTO struct {
	Ticker   string `yaml:"ticker"`
	Price    int    `yaml:"price"`
	Tradable *bool  `yaml:"tradable"`
}

// Seeds the stocks table from data/stocks.yml, the same reference data the
// memory backend reads directly at startup. Upserts by ticker, so re-running
// this against an already-seeded DB is a no-op rather than a unique-constraint
// error. No-op in memory mode -- that backend already reads the YAML itself.
func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	if cfg.DataSource == "memory" {
		log.Print("memory mode: skipping stock seed")
		return
	}

	data, err := os.ReadFile(cfg.StockPath)
	if err != nil {
		log.Fatalf("reading %s: %v", cfg.StockPath, err)
	}
	var dtos []stockDTO
	if err := yaml.Unmarshal(data, &dtos); err != nil {
		log.Fatalf("parsing %s: %v", cfg.StockPath, err)
	}

	conn, err := sql.Open(cfg.DBDriver, cfg.DBUrl)
	if err != nil {
		log.Fatalf("connecting to db: %v", err)
	}
	defer conn.Close()

	tickers := make([]string, len(dtos))
	prices := make([]int, len(dtos))
	tradables := make([]bool, len(dtos))
	for i, d := range dtos {
		tradable := true
		if d.Tradable != nil {
			tradable = *d.Tradable
		}
		tickers[i], prices[i], tradables[i] = d.Ticker, d.Price, tradable
	}

	stockRepo := postgres.NewStockRepo(*sqlc.NewStore(conn))
	seeded, err := stockRepo.UpsertBatch(ctx, tickers, prices, tradables)
	if err != nil {
		log.Fatalf("seeding stocks: %v", err)
	}
	log.Printf("seeded %d stocks from %s", len(seeded), cfg.StockPath)
}
