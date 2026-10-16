package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"app/internal/domain"
)

// TradeRepo is an in-memory stand-in for Postgres trade persistence, used when
// config.DataSource == "memory" so that mode stays fully self-contained
// without needing a live Postgres connection. Data is lost on restart.
type TradeRepo struct {
	mu     sync.Mutex
	nextID int64
	trades map[int64]domain.TradeResult
}

func NewTradeRepo() *TradeRepo {
	return &TradeRepo{trades: make(map[int64]domain.TradeResult)}
}

func (r *TradeRepo) Create(ctx context.Context, userID int64, amount int, targetPortfolio map[domain.Symbol]int, orders []domain.Order, status domain.TradeStatus) (domain.TradeResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	now := time.Now()
	trade := domain.NewTradeResult(r.nextID, amount, targetPortfolio, orders, userID, status, now, now)
	r.trades[r.nextID] = trade
	return trade, nil
}
func (r *TradeRepo) Get(ctx context.Context, id int64) (domain.TradeResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	trade, ok := r.trades[id]
	if !ok {
		return domain.TradeResult{}, fmt.Errorf("%w: id %d", domain.ErrTradeNotFound, id)
	}
	return trade, nil
}

func (r *TradeRepo) GetByUserID(ctx context.Context, userID int64) ([]domain.TradeResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	trades := []domain.TradeResult{}
	for _, trade := range r.trades {
		if trade.UserID() == userID {
			trades = append(trades, trade)
		}
	}
	return trades, nil
}
