package api

import (
	"app/config"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"app/internal/domain"

	"app/internal/repositories/memory"

	calcv1 "proto/gen"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	config       *config.Config
	router       *gin.Engine
	tradeService *domain.TradeService
}

func NewServer(config *config.Config) (*Server, error) {
	calcConn, err := grpc.NewClient(config.CalcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dialing trade-calc at %q: %w", config.CalcAddr, err)
	}
	return newServer(config, calcv1.NewCalcServiceClient(calcConn))
}

// newServer takes the calc client as a parameter (rather than dialing it
// itself) so tests can inject a fake instead of needing a live trade-calc.
func newServer(config *config.Config, calc calcv1.CalcServiceClient) (*Server, error) {
	stockRepo, sErr := memory.NewStockRepo(config.StockPath)

	portfolioRepo, pErr := memory.NewPortfolioRepo(config.PortfolioPath)
	if err := errors.Join(sErr, pErr); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	if err := validateReferentialIntegrity(stockRepo, portfolioRepo); err != nil {
		return nil, err
	}

	tradeService := domain.NewTradeService(stockRepo, portfolioRepo, calc)

	server := &Server{config: config, tradeService: tradeService}

	server.SetUpRouter()
	return server, nil
}

// validateReferentialIntegrity fails startup if any portfolio references a ticker
// absent from the stock catalogue. Both YAML files load once and never change, so
// this is a startup-time invariant, not something that should surface as a 500 on
// a customer's request. The runtime UnknownStocksInPortfolio guard in TradeService
// stays too: it's only safe to catch this at boot because the data is static today;
// if portfolios ever move to a database, tickers could be delisted at runtime and
// the request-time check becomes load-bearing again.
func validateReferentialIntegrity(stockRepo *memory.StockRepo, portfolioRepo *memory.PortfolioRepo) error {
	knownStocks, err := stockRepo.All()
	if err != nil {
		return fmt.Errorf("loading stocks for startup validation: %w", err)
	}
	portfolios, err := portfolioRepo.All()
	if err != nil {
		return fmt.Errorf("loading portfolios for startup validation: %w", err)
	}

	dangling := map[domain.Symbol]struct{}{}
	for _, p := range portfolios {
		for _, ticker := range p.Tickers() {
			if _, ok := knownStocks[ticker]; !ok {
				dangling[ticker] = struct{}{}
			}
		}
	}
	if len(dangling) > 0 {
		tickers := slices.Sorted(maps.Keys(dangling))
		return fmt.Errorf("portfolios reference unknown tickers: %v", tickers)
	}
	return nil
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
// @Tags Users
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

	res, err := server.tradeService.CreateTrade(c.Request.Context(), uri.UserID, req.Amount)
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
			Quantity: float64(o.QuantityUnits()) / float64(domain.QuantityPrecision),
		}
	}

	c.JSON(http.StatusOK, TradeResponse{
		Amount:          req.Amount,
		TargetPortfolio: res.TargetPortfolio(),
		Orders:          orders,
	})
}
