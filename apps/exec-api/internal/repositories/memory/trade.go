package memory

import (
	"context"
	"fmt"
	"sort"
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

func (r *TradeRepo) UpdateStatus(ctx context.Context, tradeId int64, status domain.TradeStatus) (domain.TradeResult, error) {
	if !status.Valid() {
		return domain.TradeResult{}, fmt.Errorf("invalid trade status %q", status)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	trade, ok := r.trades[tradeId]
	if !ok {
		return domain.TradeResult{}, fmt.Errorf("%w: id %d", domain.ErrTradeNotFound, tradeId)
	}
	updated := domain.NewTradeResult(
		trade.ID(),
		trade.Amount(),
		trade.TargetPortfolio(),
		trade.Orders(),
		trade.UserID(),
		status,
		trade.CreatedAt(),
		time.Now(),
	)
	r.trades[tradeId] = updated
	return updated, nil
}

func (r *TradeRepo) GetPaginatedByUser(ctx context.Context, userID int64, limit int64, offset int64) ([]domain.TradeResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	trades := []domain.TradeResult{}
	for _, trade := range r.trades {
		if trade.UserID() == userID {
			trades = append(trades, trade)
		}
	}
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].CreatedAt().After(trades[j].CreatedAt())
	})

	size := int64(len(trades))
	if offset >= size {
		return []domain.TradeResult{}, nil
	}
	end := offset + limit
	if end > size {
		end = size
	}
	return trades[offset:end], nil
}

func (r *TradeRepo) GetCountByUserID(ctx context.Context, userID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int64
	for _, trade := range r.trades {
		if trade.UserID() == userID {
			count++
		}
	}
	return count, nil
}
