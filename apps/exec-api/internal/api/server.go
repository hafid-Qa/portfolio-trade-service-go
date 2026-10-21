package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"app/config"
	"app/internal/domain"
	"app/internal/queue"
	"app/internal/repositories/memory"
	"app/internal/repositories/postgres"
	"app/internal/sqlc"

	calcv1 "proto/gen"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	config *config.Config
	router *gin.Engine

	tradeService *domain.TradeService
}

// portfolioCatalog is domain.PortfolioRepository plus All, which is deliberately
// excluded from that interface (the domain only ever needs one user's portfolio).
// Both memory.PortfolioRepo and postgres.PortfolioRepo satisfy it; it exists so
// validateReferentialIntegrity can enumerate every portfolio at startup without
// widening the domain-facing interface for a capability only this check needs.
type portfolioCatalog interface {
	domain.PortfolioRepository
	All(ctx context.Context) (map[int64]domain.Portfolio, error)
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
//
// config.DataSource picks the repository backend ("postgres", the default, or
// "memory" -- kept around as a reference/fallback, not a second production
// path). Both satisfy the same domain interfaces, so nothing downstream of
// this switch needs to know which one is in play.
func newServer(config *config.Config, calc calcv1.CalcServiceClient) (*Server, error) {
	var store *sqlc.Store
	var stockRepo domain.StockRepository
	var portfolioRepo portfolioCatalog
	var tradeRepo domain.TradeRepository
	var tradeProducer domain.TradeProducer
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     config.BrokerAddr,
		Username: config.BrokerUser,
		Password: config.BrokerPassword,
	})
	tradeProducer = queue.NewTradeProducer(asynqClient)

	switch config.DataSource {
	case "memory":
		var sErr, pErr error
		var memStockRepo *memory.StockRepo
		var memPortfolioRepo *memory.PortfolioRepo
		memStockRepo, sErr = memory.NewStockRepo(config.StockPath)
		memPortfolioRepo, pErr = memory.NewPortfolioRepo(config.PortfolioPath)
		if err := errors.Join(sErr, pErr); err != nil {
			return nil, fmt.Errorf("failed to initialize repositories: %w", err)
		}
		stockRepo, portfolioRepo, tradeRepo = memStockRepo, memPortfolioRepo, memory.NewTradeRepo()
		if err := validateReferentialIntegrity(context.Background(), stockRepo, portfolioRepo); err != nil {
			return nil, err
		}
	default:
		conn, err := sql.Open(config.DBDriver, config.DBUrl)
		if err != nil {
			return nil, fmt.Errorf("connecting to db: %w", err)
		}
		store = sqlc.NewStore(conn)
		stockRepo, portfolioRepo, tradeRepo = postgres.NewStockRepo(*store), postgres.NewPortfolioRepo(*store), postgres.NewTradeRepo(*store)

	}

	tradeService := domain.NewTradeService(stockRepo, portfolioRepo, tradeRepo, calc, tradeProducer)

	server := &Server{config: config, tradeService: tradeService}

	server.SetUpRouter()
	return server, nil
}

// validateReferentialIntegrity fails startup if any portfolio references a ticker
// absent from the stock catalogue. Takes the domain interfaces, not a concrete
// backend, so it works the same whether repos are Postgres-backed or (in tests)
// fakes. The runtime UnknownStocksInPortfolio guard in TradeService stays too:
// it's only safe to rely on this startup check as long as both catalogs are
// read in full here; if either ever becomes partial/paginated, the request-time
// check becomes load-bearing.
func validateReferentialIntegrity(ctx context.Context, stockRepo domain.StockRepository, portfolioRepo portfolioCatalog) error {
	knownStocks, err := stockRepo.All(ctx)
	if err != nil {
		return fmt.Errorf("loading stocks for startup validation: %w", err)
	}
	portfolios, err := portfolioRepo.All(ctx)
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
	api.POST("/users/:user_id/trades", server.TradeHandler)
	api.GET("/users/:user_id/trades/:id", server.GetTradeHanlder)
	api.GET("/users/:user_id/trades", server.GetTradeByUserHanlder)

	server.router = router
}
