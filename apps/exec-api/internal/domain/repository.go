package domain

import "context"

type StockRepository interface {
	All(ctx context.Context) (map[Symbol]Stock, error)
	GetBySymbols(ctx context.Context, symbols []Symbol) (map[Symbol]Stock, error)
}

type PortfolioRepository interface {
	Get(ctx context.Context, UserId int64) (Portfolio, error)
}

type TradeRepository interface {
	Create(ctx context.Context, userID int64, amount int, targetPortfolio map[Symbol]int, orders []Order, status TradeStatus) (TradeResult, error)
	Get(ctx context.Context, id int64) (TradeResult, error)
	GetByUserID(ctx context.Context, userID int64) ([]TradeResult, error)
	GetPaginatedByUser(ctx context.Context, userID int64, limit int64, off int64) ([]TradeResult, error)
	UpdateStatus(ctx context.Context, tradeId int64, status TradeStatus) (TradeResult, error)
	GetCountByUserID(ctx context.Context, userID int64) (int64, error)
}
