package domain

import (
	"context"

	calcv1 "proto/gen"
)

type TradeService struct {
	stockRepo     StockRepository
	PortfolioRepo PortfolioRepository
	TradeRepo     TradeRepository
	calc          calcv1.CalcServiceClient
	TradeProducer TradeProducer
}

func NewTradeService(s StockRepository, p PortfolioRepository, t TradeRepository, calc calcv1.CalcServiceClient, tP TradeProducer) *TradeService {
	return &TradeService{stockRepo: s, PortfolioRepo: p, TradeRepo: t, calc: calc, TradeProducer: tP}
}

func (s *TradeService) CreateTrade(ctx context.Context, userID int64, amount int) (TradeResult, error) {
	portfolio, err := s.PortfolioRepo.Get(ctx, userID)
	if err != nil {
		return TradeResult{}, err
	}
	tickers := portfolio.Tickers()
	stocks, err := s.stockRepo.GetBySymbols(ctx, tickers)
	if err != nil {
		return TradeResult{}, err
	}
	if len(stocks) < len(tickers) {
		missing := []string{}
		for _, sym := range tickers {
			if _, ok := stocks[sym]; !ok {
				missing = append(missing, sym.String())
			}
		}
		return TradeResult{}, UnknownStocksInPortfolio{tickers: missing}
	}

	resp, err := s.calc.Calculate(ctx, &calcv1.CalculateRequest{
		TargetPortfolio:   toInt64Map(portfolio.TargetPortfolio()),
		Stocks:            toProtoStocks(stocks),
		InvestmentAmount:  int64(amount),
		MinOrderAmount:    int64(MinOrderAmount),
		QuantityPrecision: int64(QuantityPrecision),
	})
	if err != nil {
		return TradeResult{}, err
	}
	newTrade, err := s.TradeRepo.Create(ctx, userID, amount, portfolio.TargetPortfolio(), toOrders(resp.Orders), TradeStatusPending)
	if err != nil {
		return TradeResult{}, err
	}

	// Enqueue only after the trade is durably persisted -- a job enqueued
	// before the row exists could have the worker pick it up and query for
	// a trade that isn't there yet.
	if err := s.TradeProducer.Enqueue(ctx, newTrade.ID()); err != nil {
		return TradeResult{}, err
	}
	return newTrade, nil
}

func (s *TradeService) GetTrade(ctx context.Context, userID, id int64) (TradeResult, error) {
	trade, err := s.TradeRepo.Get(ctx, id)
	if err != nil {
		return TradeResult{}, err
	}
	if trade.UserID() != userID {
		return TradeResult{}, ErrTradeNotFound
	}
	return trade, nil
}
func (s *TradeService) GetTradesByUserID(ctx context.Context, userID int64) ([]TradeResult, error) {
	trades, err := s.TradeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return []TradeResult{}, err
	}

	return trades, nil
}

func toInt64Map(m map[Symbol]int) map[string]int64 {
	result := make(map[string]int64, len(m))
	for symbol, weight := range m {
		result[symbol.String()] = int64(weight)
	}
	return result
}

func toProtoStocks(stocks map[Symbol]Stock) map[string]*calcv1.StockInfo {
	result := make(map[string]*calcv1.StockInfo, len(stocks))
	for symbol, s := range stocks {
		result[symbol.String()] = &calcv1.StockInfo{
			Price:    int64(s.Price()),
			Tradable: s.Tradable(),
		}
	}
	return result
}

func toOrders(protoOrders []*calcv1.Order) []Order {
	orders := make([]Order, len(protoOrders))
	for i, o := range protoOrders {
		orders[i] = NewOrder(Symbol(o.Symbol), int(o.Amount), int(o.QuantityUnits))
	}
	return orders
}
