package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

// findTask looks across every state a task can be in, not just "pending" --
// the "default" queue is shared with the real, live `consumer` service (this
// test runs against the same broker the dev stack uses, not an isolated test
// broker), so a task can already be active/archived/completed by the time
// this test gets around to checking for it.
func findTask(wantTradeID int64) (queue, id string, ok bool) {
	listFns := []func(string, ...asynq.ListOption) ([]*asynq.TaskInfo, error){
		testInspector.ListPendingTasks,
		testInspector.ListActiveTasks,
		testInspector.ListRetryTasks,
		testInspector.ListArchivedTasks,
		testInspector.ListCompletedTasks,
	}
	for _, list := range listFns {
		tasks, err := list("default")
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if task.Type != TypeTradeExecution {
				continue
			}
			var payload TradeExecutionPayload
			if err := json.Unmarshal(task.Payload, &payload); err != nil {
				continue
			}
			if payload.TradeID == wantTradeID {
				return "default", task.ID, true
			}
		}
	}
	return "", "", false
}

func TestTradeProducer_Enqueue(t *testing.T) {
	producer := NewTradeProducer(testClient)
	const wantTradeID = int64(999001)

	if err := producer.Enqueue(context.Background(), wantTradeID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	var (
		foundQueue, foundID string
		found               bool
	)
	for time.Now().Before(deadline) {
		if foundQueue, foundID, found = findTask(wantTradeID); found {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Fatal("Enqueue() task never appeared on the broker (checked pending/active/retry/archived/completed)")
	}

	t.Cleanup(func() {
		testInspector.DeleteTask(foundQueue, foundID)
	})
}
