package api

type TradeRequest struct {
	Amount int `json:"amount" binding:"required,numeric,gte=1000"`
}
