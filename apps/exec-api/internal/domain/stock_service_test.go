package domain

import (
	"context"
	"errors"
	"testing"
)

func TestStockService_ListStocks_SortedBySymbol(t *testing.T) {
	repo := &fakeStockRepo{stocks: map[Symbol]Stock{
		"C": mustStock(t, "C", 2222, true),
		"A": mustStock(t, "A", 1000, true),
		"E": mustStock(t, "E", 888, false),
		"B": mustStock(t, "B", 155, true),
	}}

	stocks, err := NewStockService(repo).ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}

	var got []Symbol
	for _, s := range stocks {
		got = append(got, s.Symbol())
	}
	want := []Symbol{"A", "B", "C", "E"}
	if len(got) != len(want) {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("symbols = %v, want %v", got, want)
		}
	}
	if stocks[3].Tradable() {
		t.Error("E Tradable() = true, want false (halted stocks are listed, not hidden)")
	}
}

func TestStockService_ListStocks_Empty(t *testing.T) {
	stocks, err := NewStockService(&fakeStockRepo{stocks: map[Symbol]Stock{}}).ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if stocks == nil || len(stocks) != 0 {
		t.Errorf("ListStocks() = %#v, want empty non-nil slice", stocks)
	}
}

func TestStockService_ListStocks_RepoError(t *testing.T) {
	_, err := NewStockService(&fakeStockRepo{err: errors.New("db down")}).ListStocks(context.Background())
	if err == nil {
		t.Fatal("ListStocks() error = nil, want repo error")
	}
}
