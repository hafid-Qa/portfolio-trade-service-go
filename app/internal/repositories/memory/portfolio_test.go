package memory

import (
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

	p, err := repo.Get(1)
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

	_, err = repo.Get(999)
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

	all, _ := repo.All()
	delete(all, 1)

	again, _ := repo.All()
	if _, ok := again[1]; !ok {
		t.Error("mutating the map returned by All() affected the repo's internal state")
	}
}
