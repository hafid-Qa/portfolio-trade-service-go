package postgres

import (
	"context"
	"fmt"

	"app/internal/domain"
	"app/internal/sqlc"
)

type StockRepo struct {
	store sqlc.Store
}

func NewStockRepo(store sqlc.Store) *StockRepo {
	return &StockRepo{store: store}
}

func (r *StockRepo) All(ctx context.Context) (map[domain.Symbol]domain.Stock, error) {
	rows, err := r.store.ListStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing stocks: %w", err)
	}
	return rowsToStocks(rows)
}

func (r *StockRepo) GetBySymbols(ctx context.Context, symbols []domain.Symbol) (map[domain.Symbol]domain.Stock, error) {
	tickers := make([]string, len(symbols))
	for i, s := range symbols {
		tickers[i] = s.String()
	}

	rows, err := r.store.GetStocksByTickers(ctx, tickers)
	if err != nil {
		return nil, fmt.Errorf("getting stocks by tickers: %w", err)
	}
	return rowsToStocks(rows)
}

func rowsToStocks(rows []sqlc.Stock) (map[domain.Symbol]domain.Stock, error) {
	stocks := make(map[domain.Symbol]domain.Stock, len(rows))
	for _, row := range rows {
		s, err := domain.NewStock(row.Ticker, int(row.Price), row.Tradable)
		if err != nil {
			return nil, fmt.Errorf("stock %d: %w", row.ID, err)
		}
		stocks[s.Symbol()] = s
	}
	return stocks, nil
}
