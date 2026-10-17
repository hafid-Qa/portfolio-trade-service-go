package domain

import "context"

type TradeProducer interface {
	Enqueue(ctx context.Context, tradeID int64) error
}
