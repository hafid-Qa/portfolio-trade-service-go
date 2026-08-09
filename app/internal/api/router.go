package api

import (
	"app/config"
	"errors"
	"fmt"
	"net/http"

	"app/internal/domain"

	"app/internal/repositories/memory"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	config        *config.Config
	router        *gin.Engine
	stockRepo     domain.StockRepository
	portfolioRepo domain.PortfolioRespository
}

func NewServer(config *config.Config) (*Server, error) {
	stockRepo, sErr := memory.NewStockRepo(config.StockPath)

	portfolioRepo, pErr := memory.NewPortfolioRepo(config.PortfolioPath)
	if err := errors.Join(sErr, pErr); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	server := &Server{config: config, stockRepo: stockRepo, portfolioRepo: portfolioRepo}

	server.SetUpRouter()
	return server, nil
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func (server *Server) SetUpRouter() {
	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api")
	api.GET("/health", server.healthHandler)
	api.POST("/users/:user_id/trade", server.TradeHandler)

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

// @Summary Create a trade for a user
// @Description Calculates buy orders for a user's target portfolio from a given trade amount.
// @Description The amount is apportioned across the portfolio's stocks by their target percentages;
// @Description stocks that are not tradable or whose allocation falls below the minimum order amount
// @Description are excluded and the remainder is redistributed among the rest.
// @Tags Trades
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param request body TradeRequest true "Trade amount"
// @Success 200 {object} TradeResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users/{user_id}/trade [post]
func (server *Server) TradeHandler(c *gin.Context) {
	var uri TradeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	tradeService := domain.NewTradeService(server.stockRepo, server.portfolioRepo)
	res, err := tradeService.CreateTrade(uri.UserID, req.Amount)
	if err != nil {
		if errors.Is(err, domain.ErrPortfolioNotFound) {
			c.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	orders := make([]OrderResponse, len(res.Orders()))
	for i, o := range res.Orders() {
		orders[i] = OrderResponse{
			Symbol:   o.Symbol().String(),
			Amount:   o.Amount(),
			Quantity: o.Quantity(),
		}
	}

	c.JSON(http.StatusOK, TradeResponse{
		Amount:          req.Amount,
		TargetPortfolio: res.TargetPortfolio(),
		Orders:          orders,
	})
}
