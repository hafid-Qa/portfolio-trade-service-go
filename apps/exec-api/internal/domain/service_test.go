package domain

import (
	"errors"
	"fmt"
	"testing"
)

type fakeStockRepo struct {
	stocks map[Symbol]Stock
	err    error
}

func (f *fakeStockRepo) All() (map[Symbol]Stock, error) {
	return f.stocks, f.err
}

func (f *fakeStockRepo) GetBySymbols(symbols []Symbol) (map[Symbol]Stock, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make(map[Symbol]Stock, len(symbols))
	for _, s := range symbols {
		if stock, ok := f.stocks[s]; ok {
			result[s] = stock
		}
	}
	return result, nil
}

type fakePortfolioRepo struct {
	portfolios map[int64]Portfolio
	err        error
}

func (f *fakePortfolioRepo) Get(userID int64) (Portfolio, error) {
	if f.err != nil {
		return Portfolio{}, f.err
	}
	p, ok := f.portfolios[userID]
	if !ok {
		return Portfolio{}, fmt.Errorf("%w: user %d", ErrPortfolioNotFound, userID)
	}
	return p, nil
}

func TestTradeService_CreateTrade_HappyPath(t *testing.T) {
	stockRepo := &fakeStockRepo{stocks: map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
		"B": mustStock(t, "B", 155, true),
	}}
	portfolioRepo := &fakePortfolioRepo{portfolios: map[int64]Portfolio{
		1: mustPortfolio(t, 1, map[Symbol]int{"A": 40, "B": 60}),
	}}
	svc := NewTradeService(stockRepo, portfolioRepo)

	result, err := svc.CreateTrade(1, 10000)
	if err != nil {
		t.Fatalf("CreateTrade() error = %v", err)
	}
	if result.Amount() != 10000 {
		t.Errorf("Amount() = %d, want 10000", result.Amount())
	}
	assertOrders(t, result.Orders(), []wantOrder{
		{"A", 4000, 4000},
		{"B", 6000, 38709},
	})
}

func TestTradeService_CreateTrade_PortfolioNotFound(t *testing.T) {
	stockRepo := &fakeStockRepo{stocks: map[Symbol]Stock{}}
	portfolioRepo := &fakePortfolioRepo{portfolios: map[int64]Portfolio{}}
	svc := NewTradeService(stockRepo, portfolioRepo)

	_, err := svc.CreateTrade(999, 10000)
	if !errors.Is(err, ErrPortfolioNotFound) {
		t.Errorf("CreateTrade() error = %v, want ErrPortfolioNotFound", err)
	}
}

// TestTradeService_CreateTrade_UnknownStocksInPortfolio covers the
// referential-integrity guard at the orchestration layer: a portfolio
// referencing a ticker the stock repository doesn't know about at all (as
// opposed to a ticker that's merely halted) is a data-integrity error, not
// something the calculator should silently exclude.
func TestTradeService_CreateTrade_UnknownStocksInPortfolio(t *testing.T) {
	stockRepo := &fakeStockRepo{stocks: map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
	}}
	portfolioRepo := &fakePortfolioRepo{portfolios: map[int64]Portfolio{
		1: mustPortfolio(t, 1, map[Symbol]int{"A": 50, "Z": 50}),
	}}
	svc := NewTradeService(stockRepo, portfolioRepo)

	_, err := svc.CreateTrade(1, 10000)
	var unknown UnknownStocksInPortfolio
	if !errors.As(err, &unknown) {
		t.Fatalf("CreateTrade() error = %v (%T), want UnknownStocksInPortfolio", err, err)
	}
}

func TestTradeService_CreateTrade_StockRepoError(t *testing.T) {
	stockRepo := &fakeStockRepo{err: errors.New("boom")}
	portfolioRepo := &fakePortfolioRepo{portfolios: map[int64]Portfolio{
		1: mustPortfolio(t, 1, map[Symbol]int{"A": 100}),
	}}
	svc := NewTradeService(stockRepo, portfolioRepo)

	_, err := svc.CreateTrade(1, 10000)
	if err == nil {
		t.Fatal("CreateTrade() error = nil, want the stock repo's error to propagate")
	}
}
