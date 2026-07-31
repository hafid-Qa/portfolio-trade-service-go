package main

import (
	"app/config"
	"app/docs"
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"

	ginSwagger "github.com/swaggo/gin-swagger"
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

	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	err = server.Start(fmt.Sprintf("%s:%d", config.ServerHOST, config.ServerPort))
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

type Server struct {
	config *config.Config
	router *gin.Engine
}

func NewServer(config *config.Config) (*Server, error) {

	server := &Server{config: config}

	server.SetUpRouter()
	return server, nil
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func (server *Server) SetUpRouter() {
	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api")
	api.GET("/health", server.healthHandler)

	server.router = router
}

// @Summary Health Check
// @Description Check the health of the API
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/health [get]
func (server *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
