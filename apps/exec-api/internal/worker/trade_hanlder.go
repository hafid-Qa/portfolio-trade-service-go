package worker

import (
	"app/internal/domain"
	"app/internal/queue"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

	_, err := h.tradeRepo.UpdateStatus(ctx, payload.TradeID, domain.TradeStatusExecuting)
	if err != nil {
		return fmt.Errorf("updating trade %d to executing: %w", payload.TradeID, err)
	}

	time.Sleep(time.Second)

	var status domain.TradeStatus
	if payload.TradeID%10 == 0 {
		status = domain.TradeStatusFailed
	} else {
		status = domain.TradeStatusCompleted
	}

	_, err = h.tradeRepo.UpdateStatus(ctx, payload.TradeID, status)
	if err != nil {
		return fmt.Errorf("updating trade %d to %s: %w", payload.TradeID, status, err)
	}

	return nil
}

func NewTradeHandler(tR domain.TradeRepository) *TradeHandler {
	return &TradeHandler{tradeRepo: tR}
}
