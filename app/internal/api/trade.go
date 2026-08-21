package api

import "app/internal/domain"

// TradeURI and TradeRequest are separate structs, not one, because Gin validates
// the entire struct passed to each ShouldBind* call: a single struct carrying both
// uri and json tags would fail validation on whichever half hasn't been bound yet.
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
