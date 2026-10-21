package queue

import (
	"context"
	"os"
	"testing"

	"app/config"

	"github.com/hibiken/asynq"
)

var (
	testClient    *asynq.Client
	testInspector *asynq.Inspector
)

// TestMain connects to the real broker (same one api/consumer use) rather
// than a mock -- asynq has no official in-memory broker, and this project
// already follows the same real-dependency pattern for Postgres (db_test).
// Tests are responsible for cleaning up whatever they enqueue.
func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		panic(err)
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.BrokerAddr,
		Username: cfg.BrokerUser,
		Password: cfg.BrokerPassword,
	}
	testClient = asynq.NewClient(redisOpt)
	testInspector = asynq.NewInspector(redisOpt)

	os.Exit(m.Run())
}
