package memory

import (
	"app/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

func TestNewStockRepo_Valid(t *testing.T) {
	path := writeFile(t, "stocks.yml", `
- ticker: "A"
  price: 1000
  tradable: true
- ticker: "E"
  price: 888
  tradable: false
`)

	repo, err := NewStockRepo(path)
	if err != nil {
		t.Fatalf("NewStockRepo() error = %v", err)
	}

	all, err := repo.All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("All() returned %d stocks, want 2", len(all))
	}
	if all["A"].Price() != 1000 || !all["A"].Tradable() {
		t.Errorf("A = %+v, want price=1000 tradable=true", all["A"])
	}
	if all["E"].Tradable() {
		t.Errorf("E.Tradable() = true, want false")
	}
}

// TestNewStockRepo_TradableDefaultsToTrue guards the pointer-based default: a
// row omitting `tradable` entirely must default to true, not to the zero value
// a plain `bool` field would silently take.
func TestNewStockRepo_TradableDefaultsToTrue(t *testing.T) {
	path := writeFile(t, "stocks.yml", `
- ticker: "A"
  price: 1000
`)

	repo, err := NewStockRepo(path)
	if err != nil {
		t.Fatalf("NewStockRepo() error = %v", err)
	}
	all, _ := repo.All()
	if !all["A"].Tradable() {
		t.Errorf("Tradable() = false for a row omitting the field, want true")
	}
}

func TestNewStockRepo_InvalidEntry(t *testing.T) {
	path := writeFile(t, "stocks.yml", `
- ticker: "A"
  price: -1
  tradable: true
`)

	_, err := NewStockRepo(path)
	if err == nil {
		t.Fatal("NewStockRepo() error = nil, want an error for a non-positive price")
	}
}

func TestNewStockRepo_FileNotFound(t *testing.T) {
	_, err := NewStockRepo(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	if err == nil {
		t.Fatal("NewStockRepo() error = nil, want an error for a missing file")
	}
}

func TestNewStockRepo_MalformedYAML(t *testing.T) {
	path := writeFile(t, "stocks.yml", "not: [valid, yaml: at all")
	_, err := NewStockRepo(path)
	if err == nil {
		t.Fatal("NewStockRepo() error = nil, want a parse error")
	}
}

func TestStockRepo_All_ReturnsACopy(t *testing.T) {
	path := writeFile(t, "stocks.yml", `
- ticker: "A"
  price: 1000
  tradable: true
`)
	repo, err := NewStockRepo(path)
	if err != nil {
		t.Fatalf("NewStockRepo() error = %v", err)
	}

	all, _ := repo.All()
	delete(all, "A")

	again, _ := repo.All()
	if _, ok := again["A"]; !ok {
		t.Error("mutating the map returned by All() affected the repo's internal state")
	}
}

func TestStockRepo_GetBySymbols_PartialMatch(t *testing.T) {
	path := writeFile(t, "stocks.yml", `
- ticker: "A"
  price: 1000
  tradable: true
`)
	repo, err := NewStockRepo(path)
	if err != nil {
		t.Fatalf("NewStockRepo() error = %v", err)
	}

	got, err := repo.GetBySymbols([]domain.Symbol{"A", "Z"})
	if err != nil {
		t.Fatalf("GetBySymbols() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetBySymbols() returned %d stocks, want 1 (unknown symbols silently omitted)", len(got))
	}
	if _, ok := got["A"]; !ok {
		t.Error("GetBySymbols() missing the known symbol A")
	}
}
