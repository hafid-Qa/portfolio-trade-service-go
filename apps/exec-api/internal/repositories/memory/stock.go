package memory

import (
	"app/internal/domain"
	"fmt"
	"maps"
	"os"

	"github.com/goccy/go-yaml"
)

type StockRepo struct {
	stocks map[domain.Symbol]domain.Stock
}

func (r *StockRepo) All() (map[domain.Symbol]domain.Stock, error) {
	return maps.Clone(r.stocks), nil
}
func (r *StockRepo) GetBySymbols(symbols []domain.Symbol) (map[domain.Symbol]domain.Stock, error) {
	result := make(map[domain.Symbol]domain.Stock)
	for _, sym := range symbols {
		if stock, ok := r.stocks[sym]; ok {
			result[sym] = stock
		}
	}
	return result, nil

}

type stockDTO struct {
	Ticker   string `yaml:"ticker"`
	Price    int    `yaml:"price"`
	Tradable *bool  `yaml:"tradable"`
}

func NewStockRepo(path string) (*StockRepo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var dtos []stockDTO
	if err := yaml.Unmarshal(data, &dtos); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	stocks := make(map[domain.Symbol]domain.Stock, len(dtos))
	for i, d := range dtos {
		tradable := true
		if d.Tradable != nil {
			tradable = *d.Tradable
		}
		s, err := domain.NewStock(d.Ticker, d.Price, tradable)
		if err != nil {
			return nil, fmt.Errorf("%s entry %d: %w", path, i, err)
		}
		stocks[s.Symbol()] = s
	}
	return &StockRepo{stocks: stocks}, nil
}
