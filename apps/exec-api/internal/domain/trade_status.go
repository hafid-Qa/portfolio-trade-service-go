package domain

type TradeStatus string

const (
	TradeStatusPending   TradeStatus = "pending"
	TradeStatusExecuting TradeStatus = "executing"
	TradeStatusCompleted TradeStatus = "completed"
	TradeStatusFailed    TradeStatus = "failed"
)
