package postgres

import (
	"app/internal/domain"
	"app/internal/sqlc"
	"context"
	"encoding/json"
	"fmt"
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
		return domain.TradeResult{}, fmt.Errorf("data error %s", err)
	}

	tradeTargetPortfolio, err := rawToTargetPortfolio(trade.TargetPortfolio)
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("trade %d: %w", trade.ID, err)
	}
	tradeOrders, err := rawToOrders(trade.Orders)
	if err != nil {
		return domain.TradeResult{}, fmt.Errorf("trade %d: %w", trade.ID, err)
	}

	return domain.NewTradeResult(
		int64(trade.ID),
		int(trade.Amount),
		tradeTargetPortfolio,
		tradeOrders,
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
