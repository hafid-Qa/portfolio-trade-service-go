package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	calcv1 "proto/gen"

	"calc/config"
	"calc/internal/domain"
)

type server struct {
	calcv1.UnimplementedCalcServiceServer
}

func (s *server) Calculate(ctx context.Context, req *calcv1.CalculateRequest) (*calcv1.CalculateResponse, error) {
	orders, err := domain.Calculate(
		toIntMap(req.TargetPortfolio),
		toStockMap(req.Stocks),
		int(req.InvestmentAmount),
		int(req.MinOrderAmount),
		int(req.QuantityPrecision),
	)
	if err != nil {
		return nil, err
	}
	return &calcv1.CalculateResponse{Orders: toProtoOrders(orders)}, nil
}

func toIntMap(m map[string]int64) map[string]int {
	result := make(map[string]int, len(m))
	for symbol, weight := range m {
		result[symbol] = int(weight)
	}
	return result
}

func toStockMap(stocks map[string]*calcv1.StockInfo) map[string]domain.Stock {
	result := make(map[string]domain.Stock, len(stocks))
	for symbol, s := range stocks {
		result[symbol] = domain.NewStock(symbol, int(s.Price), s.Tradable)
	}
	return result
}

func toProtoOrders(orders []domain.Order) []*calcv1.Order {
	result := make([]*calcv1.Order, len(orders))
	for i, o := range orders {
		result[i] = &calcv1.Order{
			Symbol:        o.Symbol(),
			Amount:        int64(o.Amount()),
			QuantityUnits: int64(o.QuantityUnits()),
		}
	}
	return result
}

func main() {
	cfg, err := config.LoadConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	calcv1.RegisterCalcServiceServer(grpcServer, &server{})

	log.Printf("calc gRPC server listening on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
