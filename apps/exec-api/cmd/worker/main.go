package main

import (
	"app/config"
	"context"
	"database/sql"
	"log"

	"app/internal/domain"
	"app/internal/queue"
	"app/internal/repositories/memory"
	"app/internal/repositories/postgres"
	"app/internal/sqlc"
	"app/internal/worker"

	"github.com/hibiken/asynq"
)

func main() {
	ctx := context.Background()

	config, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		panic("error while on start up")
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     config.BrokerAddr,
			Username: config.BrokerUser,
			Password: config.BrokerPassword,
		},
		asynq.Config{

			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
	var store *sqlc.Store
	var tradeRepo domain.TradeRepository

	switch config.DataSource {
	case "memory":
		tradeRepo = memory.NewTradeRepo()
	default:
		conn, err := sql.Open(config.DBDriver, config.DBUrl)
		if err != nil {
			log.Fatalf("connecting to db: %v", err)
		}
		store = sqlc.NewStore(conn)
		tradeRepo = postgres.NewTradeRepo(*store)

	}
	// mux maps a type to a handler
	mux := asynq.NewServeMux()
	mux.Handle(queue.TypeTradeExecution, worker.NewTradeHandler(tradeRepo))

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}

}
