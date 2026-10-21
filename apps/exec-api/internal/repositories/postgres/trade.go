package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"app/internal/domain"
	"app/internal/sqlc"
)

type TradeRepo struct {
	store sqlc.Store
}

func NewTradeRepo(store sqlc.Store) *TradeRepo {
	return &TradeRepo{store: store}
}

func (r *TradeRepo) Create(ctx context.Context, userID int64, amount int, targetPortfolio map[domain.Symbol]int, orders []domain.Order, status domain.TradeStatus) (domain.TradeResult, error) {
	rawPortfolio, err := portfolioToRaw(targetPortfolio)
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("data error %s", err)
	}
	rawOrder, err := ordersToRaw(orders)
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("data error %s", err)
	}
	trade, err := r.store.CreateTrade(ctx, sqlc.CreateTradeParams{
		UserID:          int32(userID),
		Amount:          int32(amount),
		TargetPortfolio: rawPortfolio,
		Orders:          rawOrder,
		Status:          string(status),
	})
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("creating trade: %w", err)
	}
	return rowToTradeResult(trade)
}

func (r *TradeRepo) Get(ctx context.Context, id int64) (domain.TradeResult, error) {
	trade, err := r.store.GetTradeByID(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TradeResult{}, fmt.Errorf("%w: id %d", domain.ErrTradeNotFound, id)
	}
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("getting trade %d: %w", id, err)
	}
	return rowToTradeResult(trade)
}

func (r *TradeRepo) GetByUserID(ctx context.Context, userID int64) ([]domain.TradeResult, error) {
	rows, err := r.store.GetTradesByUserID(ctx, int32(userID))
	if err != nil {
		return nil, fmt.Errorf("listing trades for user %d: %w", userID, err)
	}

	trades := make([]domain.TradeResult, len(rows))
	for i, row := range rows {
		trade, err := rowToTradeResult(row)
		if err != nil {
			return nil, err
		}
		trades[i] = trade
	}
	return trades, nil
}

func (r *TradeRepo) UpdateStatus(ctx context.Context, tradeId int64, status domain.TradeStatus) (domain.TradeResult, error) {
	if !status.Valid() {
		return domain.TradeResult{}, fmt.Errorf("invalid trade status %q", status)
	}
	trade, err := r.store.UpdateTradeStatus(ctx, sqlc.UpdateTradeStatusParams{
		ID:     int32(tradeId),
		Status: string(status),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TradeResult{}, fmt.Errorf("%w: id %d", domain.ErrTradeNotFound, tradeId)
	}
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("getting trade %d: %w", tradeId, err)
	}
	return rowToTradeResult(trade)
}
func rowToTradeResult(trade sqlc.Trade) (domain.TradeResult, error) {
	targetPortfolio, err := rawToTargetPortfolio(trade.TargetPortfolio)
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("trade %d: %w", trade.ID, err)
	}
	orders, err := rawToOrders(trade.Orders)
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("trade %d: %w", trade.ID, err)
	}

	return domain.NewTradeResult(
		int64(trade.ID),
		int(trade.Amount),
		targetPortfolio,
		orders,
		int64(trade.UserID),
		domain.TradeStatus(trade.Status),
		trade.CreatedAt,
		trade.UpdatedAt,
	), nil
}

func portfolioToRaw(targetPortfolio map[domain.Symbol]int) (json.RawMessage, error) {
	return json.Marshal(targetPortfolio)
}

type orderJSON struct {
	Symbol        string `json:"symbol"`
	Amount        int    `json:"amount"`
	QuantityUnits int    `json:"quantity_units"`
}

func ordersToRaw(orders []domain.Order) (json.RawMessage, error) {
	rows := make([]orderJSON, len(orders))
	for i, o := range orders {
		rows[i] = orderJSON{Symbol: o.Symbol().String(), Amount: o.Amount(), QuantityUnits: o.QuantityUnits()}
	}
	return json.Marshal(rows)
}
