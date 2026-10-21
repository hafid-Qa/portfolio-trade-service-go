package worker

import (
	"app/internal/domain"
	"app/internal/queue"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

type TradeHandler struct {
	tradeRepo domain.TradeRepository
}

func (h *TradeHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload queue.TradeExecutionPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshaling payload: %w", err)
	}
	log.Printf("processed trade execution task: trade_id=%d", payload.TradeID)
	return nil
}

func NewTradeHandler(tR domain.TradeRepository) *TradeHandler {
	return &TradeHandler{tradeRepo: tR}
}
