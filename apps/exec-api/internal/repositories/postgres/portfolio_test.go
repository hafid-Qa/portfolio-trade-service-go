package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"app/internal/domain"
	"app/internal/sqlc"
)

func seedPortfolio(t *testing.T, userID int64, weights map[string]int) {
	t.Helper()
	data, err := json.Marshal(weights)
	if err != nil {
		t.Fatalf("marshaling target_portfolio: %v", err)
	}
	_, err = testStore.CreatePortfolio(context.Background(), sqlc.CreatePortfolioParams{
		UserID:          int32(userID),
		TargetPortfolio: data,
	})
	if err != nil {
		t.Fatalf("seeding portfolio for user %d: %v", userID, err)
	}
	t.Cleanup(func() {
		testStore.DeletePortfolio(context.Background(), int32(userID))
	})
}

func TestPortfolioRepo_Get(t *testing.T) {
	seedPortfolio(t, 90001, map[string]int{"A": 40, "B": 60})

	repo := NewPortfolioRepo(testStore)
	p, err := repo.Get(context.Background(), 90001)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	target := p.TargetPortfolio()
	if target["A"] != 40 || target["B"] != 60 {
		t.Errorf("TargetPortfolio() = %v, want A:40 B:60", target)
	}
}

func TestPortfolioRepo_Get_NotFound(t *testing.T) {
	repo := NewPortfolioRepo(testStore)
	_, err := repo.Get(context.Background(), 999999)
	if !errors.Is(err, domain.ErrPortfolioNotFound) {
		t.Errorf("Get() error = %v, want ErrPortfolioNotFound", err)
	}
}

func TestPortfolioRepo_All(t *testing.T) {
	seedPortfolio(t, 90002, map[string]int{"C": 100})

	repo := NewPortfolioRepo(testStore)
	all, err := repo.All(context.Background())
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	p, ok := all[90002]
	if !ok {
		t.Fatal("All() missing seeded portfolio for user 90002")
	}
	if p.TargetPortfolio()["C"] != 100 {
		t.Errorf("TargetPortfolio() = %v, want C:100", p.TargetPortfolio())
	}
}
