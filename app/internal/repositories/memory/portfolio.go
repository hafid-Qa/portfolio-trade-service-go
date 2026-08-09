package memory

import (
	"app/internal/domain"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type PortfolioRepo struct {
	portfolios map[int64]domain.Portfolio
}

func (r *PortfolioRepo) Get(userID int64) (domain.Portfolio, error) {
	p, ok := r.portfolios[userID]
	if !ok {
		return domain.Portfolio{}, fmt.Errorf("%w: user %d", domain.ErrPortfolioNotFound, userID)
	}
	return p, nil
}

type portfolioDTO struct {
	UserId          int64          `yaml:"user_id"`
	TargetPortfolio map[string]int `yaml:"target_portfolio"`
}

func NewPortfolioRepo(path string) (*PortfolioRepo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var dtos []portfolioDTO
	if err := yaml.Unmarshal(data, &dtos); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	portfolios := make(map[int64]domain.Portfolio, len(dtos))
	for i, d := range dtos {
		// convert to symbol:int
		targetPortfolio := make(map[domain.Symbol]int, len(d.TargetPortfolio))
		for sym, ratio := range d.TargetPortfolio {
			toSymbol, err := domain.NewSymbol(sym)
			if err != nil {
				return nil, fmt.Errorf("%s entry %d: %w", path, i, err)
			}
			targetPortfolio[toSymbol] = ratio
		}

		p, err := domain.NewPortfolio(d.UserId, targetPortfolio)
		if err != nil {
			return nil, fmt.Errorf("%s entry %d: %w", path, i, err)
		}
		if _, exists := portfolios[p.UserId()]; exists {
			return nil, fmt.Errorf("%s entry %d: duplicate user id %d", path, i, p.UserId())
		}
		portfolios[p.UserId()] = p

	}
	return &PortfolioRepo{portfolios: portfolios}, nil
}
