package memory

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"app/internal/domain"
)

func TestNewPortfolioRepo_Valid(t *testing.T) {
	path := writeFile(t, "portfolio.yml", `
- user_id: 1
  target_portfolio:
    A: 40
    B: 60
- user_id: 2
  target_portfolio:
    E: 100
`)

	repo, err := NewPortfolioRepo(path)
	if err != nil {
		t.Fatalf("NewPortfolioRepo() error = %v", err)
	}

	p, err := repo.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("Get(1) error = %v", err)
	}
	target := p.TargetPortfolio()
	if target["A"] != 40 || target["B"] != 60 {
		t.Errorf("Get(1).TargetPortfolio() = %v, want A:40 B:60", target)
	}
}

func TestNewPortfolioRepo_DuplicateUserID(t *testing.T) {
	path := writeFile(t, "portfolio.yml", `
- user_id: 1
  target_portfolio:
    A: 100
- user_id: 1
  target_portfolio:
    B: 100
`)

	_, err := NewPortfolioRepo(path)
	if err == nil {
		t.Fatal("NewPortfolioRepo() error = nil, want an error for a duplicate user_id")
	}
}

func TestNewPortfolioRepo_InvalidWeights(t *testing.T) {
	path := writeFile(t, "portfolio.yml", `
- user_id: 1
  target_portfolio:
    A: 40
    B: 50
`)

	_, err := NewPortfolioRepo(path)
	if !errors.Is(err, domain.ErrInvalidPortfolioSum) {
		t.Errorf("NewPortfolioRepo() error = %v, want ErrInvalidPortfolioSum", err)
	}
}

func TestNewPortfolioRepo_FileNotFound(t *testing.T) {
	_, err := NewPortfolioRepo(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	if err == nil {
		t.Fatal("NewPortfolioRepo() error = nil, want an error for a missing file")
	}
}

func TestPortfolioRepo_Get_NotFound(t *testing.T) {
	path := writeFile(t, "portfolio.yml", `
- user_id: 1
  target_portfolio:
    A: 100
`)
	repo, err := NewPortfolioRepo(path)
	if err != nil {
		t.Fatalf("NewPortfolioRepo() error = %v", err)
	}

	_, err = repo.Get(context.Background(), 999)
	if !errors.Is(err, domain.ErrPortfolioNotFound) {
		t.Errorf("Get(999) error = %v, want ErrPortfolioNotFound", err)
	}
}

func TestPortfolioRepo_All_ReturnsACopy(t *testing.T) {
	path := writeFile(t, "portfolio.yml", `
- user_id: 1
  target_portfolio:
    A: 100
`)
	repo, err := NewPortfolioRepo(path)
	if err != nil {
		t.Fatalf("NewPortfolioRepo() error = %v", err)
	}

	all, _ := repo.All(context.Background())
	delete(all, 1)

	again, _ := repo.All(context.Background())
	if _, ok := again[1]; !ok {
		t.Error("mutating the map returned by All() affected the repo's internal state")
	}
}

func mustPortfolio(t *testing.T, userID int64, weights map[domain.Symbol]int) domain.Portfolio {
	t.Helper()
	p, err := domain.NewPortfolio(userID, weights)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPortfolioRepo_Upsert_CreatesAndReplaces(t *testing.T) {
	ctx := context.Background()
	repo, err := NewPortfolioRepo(writeFile(t, "portfolio.yml", `
- user_id: 1
  target_portfolio:
    A: 40
    B: 60
`))
	if err != nil {
		t.Fatal(err)
	}

	// New user: created.
	if _, err := repo.Upsert(ctx, mustPortfolio(t, 2, map[domain.Symbol]int{"C": 100})); err != nil {
		t.Fatalf("Upsert(new) error = %v", err)
	}
	if p, err := repo.Get(ctx, 2); err != nil || p.TargetPortfolio()["C"] != 100 {
		t.Errorf("Get(2) = %v, %v; want C:100", p.TargetPortfolio(), err)
	}

	// Existing user: replaced, not merged.
	if _, err := repo.Upsert(ctx, mustPortfolio(t, 1, map[domain.Symbol]int{"A": 100})); err != nil {
		t.Fatalf("Upsert(existing) error = %v", err)
	}
	p, _ := repo.Get(ctx, 1)
	if w := p.TargetPortfolio(); len(w) != 1 || w["A"] != 100 {
		t.Errorf("Get(1) = %v, want only A:100", w)
	}
}
