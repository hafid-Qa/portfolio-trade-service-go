package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// TypeTradeExecution is the asynq task type for "go execute this trade" --
// shared between the producer (here) and the worker's handler, so the two
// sides never duplicate the string literal.
const TypeTradeExecution = "trade:execute"

// TradeExecutionPayload carries only the trade's ID, not its data -- the
// database is the source of truth, so the worker re-fetches current state
// via TradeRepository.Get when it processes the task, rather than trusting
// a snapshot that could already be stale by the time the job runs.
type TradeExecutionPayload struct {
	TradeID int64 `json:"trade_id"`
}

type TradeProducer struct {
	client *asynq.Client
}

func NewTradeProducer(client *asynq.Client) *TradeProducer {
	return &TradeProducer{client: client}
}

func (p *TradeProducer) Enqueue(ctx context.Context, tradeID int64) error {
	payload, err := json.Marshal(TradeExecutionPayload{TradeID: tradeID})
	if err != nil {
		return fmt.Errorf("marshaling trade execution payload: %w", err)
	}

	task := asynq.NewTask(TypeTradeExecution, payload)
	if _, err := p.client.EnqueueContext(ctx, task); err != nil {
		return fmt.Errorf("enqueueing trade %d: %w", tradeID, err)
	}
	return nil
}
