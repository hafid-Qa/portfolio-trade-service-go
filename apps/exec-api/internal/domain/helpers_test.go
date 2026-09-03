package domain

import "testing"

func mustStock(t *testing.T, ticker string, price int, tradable bool) Stock {
	t.Helper()
	s, err := NewStock(ticker, price, tradable)
	if err != nil {
		t.Fatalf("NewStock(%q, %d, %v) failed: %v", ticker, price, tradable, err)
	}
	return s
}

func mustPortfolio(t *testing.T, userID int64, weights map[Symbol]int) Portfolio {
	t.Helper()
	p, err := NewPortfolio(userID, weights)
	if err != nil {
		t.Fatalf("NewPortfolio(%d, %v) failed: %v", userID, weights, err)
	}
	return p
}

type wantOrder struct {
	symbol        Symbol
	amount        int
	quantityUnits int
}

func assertOrders(t *testing.T, got []Order, want []wantOrder) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d orders, want %d: got=%+v want=%+v", len(got), len(want), got, want)
	}
	for i, w := range want {
		g := got[i]
		if g.Symbol() != w.symbol || g.Amount() != w.amount || g.QuantityUnits() != w.quantityUnits {
			t.Errorf("order %d = {%s %d %d}, want {%s %d %d}",
				i, g.Symbol(), g.Amount(), g.QuantityUnits(), w.symbol, w.amount, w.quantityUnits)
		}
	}
}
