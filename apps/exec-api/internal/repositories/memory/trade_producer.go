package memory

import (
	"context"
	"log"
)

// TradeProducer is a no-op stand-in for the asynq-backed producer, used when
// config.DataSource == "memory" so that mode stays fully self-contained
// without needing a live broker connection. Enqueued trades are never
// actually processed -- there's no worker running against memory data.
type TradeProducer struct{}

func NewTradeProducer() *TradeProducer {
	return &TradeProducer{}
}

func (p *TradeProducer) Enqueue(ctx context.Context, tradeID int64) error {
	log.Printf("memory mode: skipping enqueue for trade %d (no broker)", tradeID)
	return nil
}
