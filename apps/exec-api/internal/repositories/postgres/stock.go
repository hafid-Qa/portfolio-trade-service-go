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

// UpsertBatch inserts stocks or, for tickers that already exist, updates
// their price and tradability -- all in one round trip. Used by cmd/seed to
// make loading data/stocks.yml idempotent across repeated `docker compose
// up`s on an existing DB.
func (r *StockRepo) UpsertBatch(ctx context.Context, tickers []string, prices []int, tradables []bool) ([]domain.Stock, error) {
	int32Prices := make([]int32, len(prices))
	for i, p := range prices {
		int32Prices[i] = int32(p)
	}
	rows, err := r.store.UpsertStocks(ctx, sqlc.UpsertStocksParams{
		Tickers:   tickers,
		Prices:    int32Prices,
		Tradables: tradables,
	})
	if err != nil {
		return nil, fmt.Errorf("upserting %d stocks: %w", len(tickers), err)
	}
	stocks := make([]domain.Stock, len(rows))
	for i, row := range rows {
		s, err := domain.NewStock(row.Ticker, int(row.Price), row.Tradable)
		if err != nil {
			return nil, fmt.Errorf("stock %d: %w", row.ID, err)
		}
		stocks[i] = s
	}
	return stocks, nil
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
