package domain

import (
	"context"

	calcv1 "proto/gen"
)

type TradeService struct {
	stockRepo     StockRepository
	PortfolioRepo PortfolioRepository
	calc          calcv1.CalcServiceClient
}

func NewTradeService(s StockRepository, p PortfolioRepository, calc calcv1.CalcServiceClient) *TradeService {
	return &TradeService{stockRepo: s, PortfolioRepo: p, calc: calc}
}

func (s *TradeService) CreateTrade(ctx context.Context, userID int64, amount int) (TradeResult, error) {
	portfolio, err := s.PortfolioRepo.Get(userID)
	if err != nil {
		return TradeResult{}, err
	}
	tickers := portfolio.Tickers()
	stocks, err := s.stockRepo.GetBySymbols(tickers)
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

	return NewTradeResult(amount, portfolio.TargetPortfolio(), toOrders(resp.Orders)), nil
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
