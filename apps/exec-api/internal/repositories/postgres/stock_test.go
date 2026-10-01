package postgres

import (
	"context"
	"testing"

	"app/internal/db/sqlc"
	"app/internal/domain"
)

func seedStock(t *testing.T, ticker string, price int, tradable bool) {
	t.Helper()
	_, err := testStore.CreateStock(context.Background(), sqlc.CreateStockParams{
		Ticker:   ticker,
		Price:    int32(price),
		Tradable: tradable,
	})
	if err != nil {
		t.Fatalf("seeding stock %s: %v", ticker, err)
	}
	t.Cleanup(func() {
		testStore.DeleteStock(context.Background(), ticker)
	})
}

func TestStockRepo_All(t *testing.T) {
	seedStock(t, "SA1", 1000, true)
	seedStock(t, "SA2", 155, false)

	repo := NewStockRepo(testStore)
	all, err := repo.All(context.Background())
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}

	a, ok := all["SA1"]
	if !ok {
		t.Fatal("All() missing seeded stock SA1")
	}
	if a.Price() != 1000 || !a.Tradable() {
		t.Errorf("SA1 = %+v, want price=1000 tradable=true", a)
	}

	b, ok := all["SA2"]
	if !ok {
		t.Fatal("All() missing seeded stock SA2")
	}
	if b.Tradable() {
		t.Errorf("SA2.Tradable() = true, want false")
	}
}

func TestStockRepo_GetBySymbols_PartialMatch(t *testing.T) {
	seedStock(t, "SG1", 2000, true)

	repo := NewStockRepo(testStore)
	got, err := repo.GetBySymbols(context.Background(), []domain.Symbol{"SG1", "SGX"})
	if err != nil {
		t.Fatalf("GetBySymbols() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetBySymbols() returned %d stocks, want 1 (unknown symbols silently omitted)", len(got))
	}
	if _, ok := got["SG1"]; !ok {
		t.Error("GetBySymbols() missing the known symbol SG1")
	}
}
