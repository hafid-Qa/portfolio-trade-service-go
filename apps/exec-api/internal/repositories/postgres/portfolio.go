package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"app/internal/db/sqlc"
	"app/internal/domain"
)

type PortfolioRepo struct {
	store sqlc.Store
}

func NewPortfolioRepo(store sqlc.Store) *PortfolioRepo {
	return &PortfolioRepo{store: store}
}

func (r *PortfolioRepo) Get(ctx context.Context, userID int64) (domain.Portfolio, error) {
	row, err := r.store.GetPortfolioByUserID(ctx, int32(userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Portfolio{}, fmt.Errorf("%w: user %d", domain.ErrPortfolioNotFound, userID)
	}
	if err != nil {
		return domain.Portfolio{}, fmt.Errorf("getting portfolio for user %d: %w", userID, err)
	}
	return rowToPortfolio(row)
}

// All is deliberately not part of domain.PortfolioRepository: the domain only ever
// needs one user's portfolio. This exists solely for the startup referential-integrity
// check in api.NewServer, which needs to enumerate every portfolio before the app
// starts serving requests.
func (r *PortfolioRepo) All(ctx context.Context) (map[int64]domain.Portfolio, error) {
	rows, err := r.store.ListPortfolios(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing portfolios: %w", err)
	}

	portfolios := make(map[int64]domain.Portfolio, len(rows))
	for _, row := range rows {
		p, err := rowToPortfolio(row)
		if err != nil {
			return nil, err
		}
		portfolios[p.UserId()] = p
	}
	return portfolios, nil
}

func rowToPortfolio(row sqlc.Portfolio) (domain.Portfolio, error) {
	var weights map[string]int
	if err := json.Unmarshal(row.TargetPortfolio, &weights); err != nil {
		return domain.Portfolio{}, fmt.Errorf("portfolio %d: parsing target_portfolio: %w", row.ID, err)
	}

	targetPortfolio := make(map[domain.Symbol]int, len(weights))
	for sym, ratio := range weights {
		symbol, err := domain.NewSymbol(sym)
		if err != nil {
			return domain.Portfolio{}, fmt.Errorf("portfolio %d: %w", row.ID, err)
		}
		targetPortfolio[symbol] = ratio
	}

	return domain.NewPortfolio(int64(row.UserID), targetPortfolio)

}
