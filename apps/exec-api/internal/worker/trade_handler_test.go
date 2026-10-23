package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"app/internal/domain"
	"app/internal/queue"
	"app/internal/repositories/memory"

	"github.com/hibiken/asynq"
)

func newTradeTask(t *testing.T, tradeID int64) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(queue.TradeExecutionPayload{TradeID: tradeID})
	if err != nil {
		t.Fatalf("marshaling payload: %v", err)
	}
	return asynq.NewTask(queue.TypeTradeExecution, payload)
}

func seedTrade(t *testing.T, repo *memory.TradeRepo, userID int64) domain.TradeResult {
	t.Helper()
	trade, err := repo.Create(context.Background(), userID, 100, map[domain.Symbol]int{}, nil, domain.TradeStatusPending)
	if err != nil {
		t.Fatalf("seeding trade: %v", err)
	}
	return trade
}

func TestTradeHandler_ProcessTask_Completed(t *testing.T) {
	repo := memory.NewTradeRepo()
	var trade domain.TradeResult
	for {
		trade = seedTrade(t, repo, 1)
		if trade.ID()%10 != 0 {
			break
		}
	}

	handler := NewTradeHandler(repo)
	if err := handler.ProcessTask(context.Background(), newTradeTask(t, trade.ID())); err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}

	got, err := repo.Get(context.Background(), trade.ID())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status() != domain.TradeStatusCompleted {
		t.Fatalf("Status() = %q, want %q", got.Status(), domain.TradeStatusCompleted)
	}
}

func TestTradeHandler_ProcessTask_Failed(t *testing.T) {
	repo := memory.NewTradeRepo()
	var trade domain.TradeResult
	for {
		trade = seedTrade(t, repo, 1)
		if trade.ID()%10 == 0 {
			break
		}
	}

	handler := NewTradeHandler(repo)
	if err := handler.ProcessTask(context.Background(), newTradeTask(t, trade.ID())); err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}

	got, err := repo.Get(context.Background(), trade.ID())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status() != domain.TradeStatusFailed {
		t.Fatalf("Status() = %q, want %q", got.Status(), domain.TradeStatusFailed)
	}
}

func TestTradeHandler_ProcessTask_InvalidPayload(t *testing.T) {
	repo := memory.NewTradeRepo()
	handler := NewTradeHandler(repo)
	task := asynq.NewTask(queue.TypeTradeExecution, []byte("not json"))

	err := handler.ProcessTask(context.Background(), task)
	if err == nil {
		t.Fatal("ProcessTask() error = nil, want error for invalid payload")
	}
	if !errors.Is(err, asynq.SkipRetry) {
		t.Errorf("ProcessTask() error = %v, want asynq.SkipRetry (malformed payload can't be fixed by retrying)", err)
	}
}

func TestTradeHandler_ProcessTask_UnknownTrade(t *testing.T) {
	repo := memory.NewTradeRepo()
	handler := NewTradeHandler(repo)

	err := handler.ProcessTask(context.Background(), newTradeTask(t, 999999))
	if err == nil {
		t.Fatal("ProcessTask() error = nil, want error for unknown trade id")
	}
	if !errors.Is(err, asynq.SkipRetry) {
		t.Errorf("ProcessTask() error = %v, want asynq.SkipRetry (unknown trade ID can't be fixed by retrying)", err)
	}
}

func TestTradeHandler_ProcessTask_SetsExecutingBeforeSleeping(t *testing.T) {
	repo := memory.NewTradeRepo()
	trade := seedTrade(t, repo, 1)
	handler := NewTradeHandler(repo)

	done := make(chan error, 1)
	go func() {
		done <- handler.ProcessTask(context.Background(), newTradeTask(t, trade.ID()))
	}()

	time.Sleep(100 * time.Millisecond)
	got, err := repo.Get(context.Background(), trade.ID())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status() != domain.TradeStatusExecuting {
		t.Fatalf("Status() while processing = %q, want %q", got.Status(), domain.TradeStatusExecuting)
	}

	if err := <-done; err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}
}
