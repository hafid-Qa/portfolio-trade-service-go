package main

import (
	"app/config"
	"app/docs"
	"app/internal/api"
	"context"
	"fmt"
	"log"
)

// @title Portfolio Trade Service API
// @version 1.0
// @description API for Trade Service
// @BasePath /
// @securityDefinitions.oauth2.password OAuth2Password
// @tokenUrl /api/users/login
// @description Log in with username and password; the token is then sent as
// @description "Authorization: Bearer <token>" on every request.
func main() {
	ctx := context.Background()

	config, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		panic("error while on start up")
	}

	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", config.ExternalPort)

	server, err := api.NewServer(config)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	err = server.Start(fmt.Sprintf("%s:%d", config.ServerHOST, config.ServerPort))
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
