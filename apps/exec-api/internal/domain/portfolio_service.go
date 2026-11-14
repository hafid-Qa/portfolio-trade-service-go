package domain

import (
	"context"
	"slices"
)

type PortfolioService struct {
	stockRepo     StockRepository
	portfolioRepo PortfolioRepository
}

func NewPortfolioService(stockRepo StockRepository, portfolioRepo PortfolioRepository) *PortfolioService {
	return &PortfolioService{stockRepo: stockRepo, portfolioRepo: portfolioRepo}
}

// SetPortfolio validates and stores the user's target portfolio, replacing
// any previous one. Validation errors match ErrInvalidPortfolio,
// ErrInvalidPortfolioSum or UnknownStocksInPortfolio.
func (s *PortfolioService) SetPortfolio(ctx context.Context, userID int64, targetPortfolio map[Symbol]int) (Portfolio, error) {
	p, err := NewPortfolio(userID, targetPortfolio)
	if err != nil {
		return Portfolio{}, err
	}

	// Reject unknown tickers now, rather than letting every later trade fail.
	tickers := p.Tickers()
	stocks, err := s.stockRepo.GetBySymbols(ctx, tickers)
	if err != nil {
		return Portfolio{}, err
	}
	if len(stocks) < len(tickers) {
		missing := []string{}
		for _, sym := range tickers {
			if _, ok := stocks[sym]; !ok {
				missing = append(missing, sym.String())
			}
		}
		slices.Sort(missing)
		return Portfolio{}, UnknownStocksInPortfolio{tickers: missing}
	}

	return s.portfolioRepo.Upsert(ctx, p)
}

// GetPortfolio returns the user's portfolio, or an error matching ErrPortfolioNotFound.
func (s *PortfolioService) GetPortfolio(ctx context.Context, userID int64) (Portfolio, error) {
	return s.portfolioRepo.Get(ctx, userID)
}
