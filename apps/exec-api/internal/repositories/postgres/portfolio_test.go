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
	seedUser(t, userID)
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

func TestPortfolioRepo_Upsert(t *testing.T) {
	ctx := context.Background()
	const userID = 90010
	seedUser(t, userID)
	t.Cleanup(func() { testStore.DeletePortfolio(ctx, userID) })
	repo := NewPortfolioRepo(testStore)

	first, err := domain.NewPortfolio(userID, map[domain.Symbol]int{"A": 40, "B": 60})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("Upsert(create) error = %v", err)
	}

	second, err := domain.NewPortfolio(userID, map[domain.Symbol]int{"C": 100})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := repo.Upsert(ctx, second)
	if err != nil {
		t.Fatalf("Upsert(replace) error = %v", err)
	}
	if w := saved.TargetPortfolio(); len(w) != 1 || w["C"] != 100 {
		t.Errorf("Upsert(replace) = %v, want only C:100", w)
	}

	// Still exactly one row for the user (the UNIQUE constraint + ON CONFLICT).
	rows, err := testStore.ListPortfolios(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range rows {
		if r.UserID == userID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("portfolio rows for user = %d, want 1", count)
	}
}

func TestPortfolioRepo_Upsert_UnknownUser(t *testing.T) {
	p, err := domain.NewPortfolio(999_999, map[domain.Symbol]int{"A": 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewPortfolioRepo(testStore).Upsert(context.Background(), p); err == nil {
		t.Fatal("Upsert() error = nil, want foreign key violation for a non-existent user")
	}
}

// The UNIQUE constraint itself, independent of the upsert query.
func TestPortfolios_UniquePerUser(t *testing.T) {
	seedPortfolio(t, 90011, map[string]int{"A": 100})
	_, err := testStore.CreatePortfolio(context.Background(), sqlc.CreatePortfolioParams{
		UserID:          90011,
		TargetPortfolio: []byte(`{"B": 100}`),
	})
	if err == nil {
		t.Fatal("second CreatePortfolio for the same user succeeded, want unique violation")
	}
}
