package postgres

import (
	"encoding/json"
	"fmt"

	"app/internal/domain"
)

func rawToTargetPortfolio(raw json.RawMessage) (map[domain.Symbol]int, error) {
	var weights map[string]int
	if err := json.Unmarshal(raw, &weights); err != nil {
		return nil, fmt.Errorf("parsing target_portfolio: %w", err)
	}

	targetPortfolio := make(map[domain.Symbol]int, len(weights))
	for sym, ratio := range weights {
		symbol, err := domain.NewSymbol(sym)
		if err != nil {
			return nil, err
		}
		targetPortfolio[symbol] = ratio
	}
	return targetPortfolio, nil
}

func rawToOrders(raw json.RawMessage) ([]domain.Order, error) {
	var rows []orderJSON
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("parsing orders: %w", err)
	}

	orders := make([]domain.Order, len(rows))
	for i, row := range rows {
		symbol, err := domain.NewSymbol(row.Symbol)
		if err != nil {
			return nil, err
		}
		orders[i] = domain.NewOrder(symbol, row.Amount, row.QuantityUnits)
	}
	return orders, nil
}
