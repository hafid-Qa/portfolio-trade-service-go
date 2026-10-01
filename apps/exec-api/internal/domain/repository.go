package domain

import "context"

type StockRepository interface {
	All(ctx context.Context) (map[Symbol]Stock, error)
	GetBySymbols(ctx context.Context, symbols []Symbol) (map[Symbol]Stock, error)
}

type PortfolioRepository interface {
	Get(ctx context.Context, UserId int64) (Portfolio, error)
}
