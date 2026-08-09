package api

import "app/internal/domain"

type TradeURI struct {
	UserID int64 `uri:"user_id" binding:"required,gte=1"`
}

type TradeRequest struct {
	Amount int `json:"amount" binding:"required,numeric,gte=1000"`
}

type OrderResponse struct {
	Symbol   string  `json:"symbol"`
	Amount   int     `json:"amount"`
	Quantity float64 `json:"quantity"`
}

type TradeResponse struct {
	Amount          int                   `json:"amount"`
	TargetPortfolio map[domain.Symbol]int `json:"target_portfolio"`
	Orders          []OrderResponse       `json:"orders"`
}
