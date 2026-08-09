package main

import (
	"app/config"
	"app/docs"
	"app/internal/api"
	"context"
	"fmt"
	"log"
)

// @title Image Pipeline API
// @version 1.0
// @description API for image pipeline services
// @BasePath /
func main() {
	ctx := context.Background()

	config, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		panic("error while on start up")
	}

	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", config.ServerPort)

	server, err := api.NewServer(config)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	err = server.Start(fmt.Sprintf("%s:%d", config.ServerHOST, config.ServerPort))
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
