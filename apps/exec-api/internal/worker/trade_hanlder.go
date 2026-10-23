package worker

import (
	"app/internal/domain"
	"app/internal/queue"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

type TradeHandler struct {
	tradeRepo domain.TradeRepository
}

// terminal wraps err with asynq.SkipRetry -- used for failures that retrying
// can never fix (malformed payload, a trade ID that doesn't exist), so asynq
// archives the task immediately instead of burning retries on it.
func terminal(err error) error {
	return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
}

func (h *TradeHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload queue.TradeExecutionPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return terminal(fmt.Errorf("unmarshaling payload: %w", err))
	}
	log.Printf("processed trade execution task: trade_id=%d", payload.TradeID)

	_, err := h.tradeRepo.UpdateStatus(ctx, payload.TradeID, domain.TradeStatusExecuting)
	if err != nil {
		err = fmt.Errorf("updating trade %d to executing: %w", payload.TradeID, err)
		if errors.Is(err, domain.ErrTradeNotFound) {
			return terminal(err)
		}
		return err
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
		err = fmt.Errorf("updating trade %d to %s: %w", payload.TradeID, status, err)
		if errors.Is(err, domain.ErrTradeNotFound) {
			return terminal(err)
		}
		return err
	}

	return nil
}

func NewTradeHandler(tR domain.TradeRepository) *TradeHandler {
	return &TradeHandler{tradeRepo: tR}
}
