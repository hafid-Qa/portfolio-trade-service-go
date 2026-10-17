package main

import (
	"app/config"
	"context"
	"log"

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
		asynq.RedisClientOpt{Addr: config.BrokerAddr},
		asynq.Config{

			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	// mux maps a type to a handler
	mux := asynq.NewServeMux()

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}

}
