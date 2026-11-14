package domain

import (
	"cmp"
	"context"
	"maps"
	"slices"
)

type StockService struct {
	stockRepo StockRepository
}

func NewStockService(stockRepo StockRepository) *StockService {
	return &StockService{stockRepo: stockRepo}
}

// ListStocks returns the whole catalogue sorted by symbol, so the order is
// stable across calls (the repository returns a map).
func (s *StockService) ListStocks(ctx context.Context) ([]Stock, error) {
	stocks, err := s.stockRepo.All(ctx)
	if err != nil {
		return nil, err
	}
	sorted := slices.SortedFunc(maps.Values(stocks), func(a, b Stock) int {
		return cmp.Compare(a.Symbol(), b.Symbol())
	})
	if sorted == nil {
		sorted = []Stock{} // an empty catalogue is [] in JSON, not null
	}
	return sorted, nil
}
