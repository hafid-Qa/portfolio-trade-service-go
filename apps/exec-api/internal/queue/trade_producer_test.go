package queue

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTradeProducer_Enqueue(t *testing.T) {
	producer := NewTradeProducer(testClient)
	const wantTradeID = int64(999001)

	if err := producer.Enqueue(context.Background(), wantTradeID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	tasks, err := testInspector.ListPendingTasks("default")
	if err != nil {
		t.Fatalf("ListPendingTasks() error = %v", err)
	}

	var found *string
	for _, task := range tasks {
		if task.Type != TypeTradeExecution {
			continue
		}
		var payload TradeExecutionPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			continue
		}
		if payload.TradeID == wantTradeID {
			id := task.ID
			found = &id
			break
		}
	}
	if found == nil {
		t.Fatal("Enqueue() did not produce a pending task carrying the expected trade ID")
	}

	t.Cleanup(func() {
		testInspector.DeleteTask("default", *found)
	})
}
