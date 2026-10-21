package domain

type TradeStatus string

const (
	TradeStatusPending   TradeStatus = "pending"
	TradeStatusExecuting TradeStatus = "executing"
	TradeStatusCompleted TradeStatus = "completed"
	TradeStatusFailed    TradeStatus = "failed"
)

func (s TradeStatus) Valid() bool {
	switch s {
	case TradeStatusPending, TradeStatusExecuting, TradeStatusCompleted, TradeStatusFailed:
		return true
	default:
		return false
	}
}
