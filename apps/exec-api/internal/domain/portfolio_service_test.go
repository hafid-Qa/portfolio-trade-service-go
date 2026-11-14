package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func newTestPortfolioService(t *testing.T, portfolios *fakePortfolioRepo, stockErr error) *PortfolioService {
	t.Helper()
	stocks := &fakeStockRepo{
		stocks: map[Symbol]Stock{
			"A": mustStock(t, "A", 1000, true),
			"B": mustStock(t, "B", 155, true),
		},
		err: stockErr,
	}
	return NewPortfolioService(stocks, portfolios)
}

func TestPortfolioService_SetPortfolio_Creates(t *testing.T) {
	repo := &fakePortfolioRepo{}
	svc := newTestPortfolioService(t, repo, nil)

	p, err := svc.SetPortfolio(context.Background(), 5, map[Symbol]int{"A": 40, "B": 60})
	if err != nil {
		t.Fatalf("SetPortfolio() error = %v", err)
	}
	if p.UserId() != 5 || p.TargetPortfolio()["A"] != 40 || p.TargetPortfolio()["B"] != 60 {
		t.Errorf("returned %+v", p.TargetPortfolio())
	}
	stored, err := svc.GetPortfolio(context.Background(), 5)
	if err != nil || stored.TargetPortfolio()["B"] != 60 {
		t.Errorf("GetPortfolio() = %v, %v; want stored portfolio", stored.TargetPortfolio(), err)
	}
}

func TestPortfolioService_SetPortfolio_Replaces(t *testing.T) {
	repo := &fakePortfolioRepo{}
	svc := newTestPortfolioService(t, repo, nil)
	ctx := context.Background()

	if _, err := svc.SetPortfolio(ctx, 5, map[Symbol]int{"A": 40, "B": 60}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetPortfolio(ctx, 5, map[Symbol]int{"A": 100}); err != nil {
		t.Fatal(err)
	}

	got, _ := svc.GetPortfolio(ctx, 5)
	if w := got.TargetPortfolio(); len(w) != 1 || w["A"] != 100 {
		t.Errorf("portfolio = %v, want only A:100 (replaced, not merged)", w)
	}
}

func TestPortfolioService_SetPortfolio_Invalid(t *testing.T) {
	cases := map[string]struct {
		weights map[Symbol]int
		check   func(error) bool
	}{
		"sum not 100": {map[Symbol]int{"A": 40, "B": 50}, func(err error) bool { return errors.Is(err, ErrInvalidPortfolioSum) }},
		"negative":    {map[Symbol]int{"A": 150, "B": -50}, func(err error) bool { return errors.Is(err, ErrInvalidPortfolio) }},
		"empty":       {map[Symbol]int{}, func(err error) bool { return errors.Is(err, ErrInvalidPortfolioSum) }},
		"unknown stock": {map[Symbol]int{"A": 50, "ZZZ": 30, "YYY": 20}, func(err error) bool {
			var u UnknownStocksInPortfolio
			return errors.As(err, &u) && strings.Contains(err.Error(), "YYY, ZZZ") // sorted
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakePortfolioRepo{}
			svc := newTestPortfolioService(t, repo, nil)

			_, err := svc.SetPortfolio(context.Background(), 5, c.weights)
			if !c.check(err) {
				t.Fatalf("SetPortfolio() error = %v", err)
			}
			if _, ok := repo.portfolios[5]; ok {
				t.Error("invalid portfolio was stored")
			}
		})
	}
}

func TestPortfolioService_SetPortfolio_StockRepoError(t *testing.T) {
	repo := &fakePortfolioRepo{}
	svc := newTestPortfolioService(t, repo, errors.New("db down"))

	if _, err := svc.SetPortfolio(context.Background(), 5, map[Symbol]int{"A": 100}); err == nil {
		t.Fatal("SetPortfolio() error = nil, want stock repo error")
	}
	if _, ok := repo.portfolios[5]; ok {
		t.Error("portfolio stored despite failed ticker check")
	}
}

func TestPortfolioService_GetPortfolio_NotFound(t *testing.T) {
	svc := newTestPortfolioService(t, &fakePortfolioRepo{}, nil)
	if _, err := svc.GetPortfolio(context.Background(), 5); !errors.Is(err, ErrPortfolioNotFound) {
		t.Errorf("GetPortfolio() error = %v, want ErrPortfolioNotFound", err)
	}
}
