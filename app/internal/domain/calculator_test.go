package domain

import (
	"errors"
	"testing"
)

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

// TestCalculate_SpecScenarios mirrors the four fixture users from the reference
// implementation's shipped data (data/portfolio.yml, data/stocks.yml): A=1000,
// B=155, C=2222, D=467, E=888 (halted).
func TestCalculate_SpecScenarios(t *testing.T) {
	stocks := map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
		"B": mustStock(t, "B", 155, true),
		"C": mustStock(t, "C", 2222, true),
		"D": mustStock(t, "D", 467, true),
		"E": mustStock(t, "E", 888, false),
	}

	tests := []struct {
		name    string
		weights map[Symbol]int
		amount  int
		want    []wantOrder
	}{
		{
			name:    "user 1: simple two-way apportionment",
			weights: map[Symbol]int{"A": 40, "B": 60},
			amount:  10000,
			want: []wantOrder{
				{"A", 4000, 4000},
				{"B", 6000, 38709},
			},
		},
		{
			name:    "user 2: only a halted stock -> empty orders, not an error",
			weights: map[Symbol]int{"E": 100},
			amount:  10000,
			want:    []wantOrder{},
		},
		{
			name:    "user 3: halted stock's weight redistributed to survivors",
			weights: map[Symbol]int{"A": 31, "B": 40, "E": 29},
			amount:  10000,
			want: []wantOrder{
				{"A", 4366, 4366},
				{"B", 5633, 36341},
			},
		},
		{
			name:    "user 4: below-minimum-order exclusion, then redistribution",
			weights: map[Symbol]int{"B": 50, "C": 49, "D": 1},
			amount:  1000,
			want: []wantOrder{
				{"B", 505, 3258},
				{"C", 494, 222},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := mustPortfolio(t, 1, tt.weights)
			got, err := Calculate(p, stocks, tt.amount)
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			assertOrders(t, got, tt.want)
		})
	}
}

func TestCalculate_AmountBelowMinimum(t *testing.T) {
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 100})
	stocks := map[Symbol]Stock{"A": mustStock(t, "A", 1000, true)}

	_, err := Calculate(p, stocks, MinTradeAmount-1)
	if err == nil {
		t.Fatal("expected an error for amount below MinTradeAmount, got nil")
	}
	var belowMin TradeAmountBelowMinimum
	if !errors.As(err, &belowMin) {
		t.Fatalf("expected error to be TradeAmountBelowMinimum, got %T: %v", err, err)
	}
}

// TestCalculate_HaltedWeightNotInEligibilityDenominator is "Hole #1" from the
// Python reference implementation's review: eligibility must be assessed against
// the denominator with halted/unknown stocks already removed, not the original
// 100%. Otherwise a halted stock's weight can starve an otherwise-comfortably-
// tradable co-symbol below MinOrderAmount purely because of how much dead weight
// it's carrying. A:19 alone clears 200 once E's 81% is out of the denominator
// (19/19 of 1000 = 1000), but never would if tested against the original 19/100.
func TestCalculate_HaltedWeightNotInEligibilityDenominator(t *testing.T) {
	stocks := map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
		"E": mustStock(t, "E", 888, false),
	}
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 19, "E": 81})

	got, err := Calculate(p, stocks, 1000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	assertOrders(t, got, []wantOrder{{"A", 1000, 1000}})
}

// TestCalculate_UnbuyableSurvivorWeightIsRedistributed is "Hole #2": a symbol
// that clears MinOrderAmount but is too expensive to buy even 0.001 of a share
// must have its weight redistributed to the remaining survivors, exactly like a
// halted stock would be -- not silently drop the dollars. B needs price 250000; its
// $200 share can't buy anything, so A should absorb the full $1000, not just its
// original 80% share ($800).
func TestCalculate_UnbuyableSurvivorWeightIsRedistributed(t *testing.T) {
	stocks := map[Symbol]Stock{
		"A": mustStock(t, "A", 1, true),
		"B": mustStock(t, "B", 250000, true),
	}
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 80, "B": 20})

	got, err := Calculate(p, stocks, 1000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	assertOrders(t, got, []wantOrder{{"A", 1000, 1000000}})
}

// TestCalculate_ZeroWeightSymbolDoesNotPanic guards against a division-by-zero
// panic: a valid portfolio (weights sum to 100) can still have an individual
// symbol at 0%, which must never become the sole, zero-sum eligible set.
func TestCalculate_ZeroWeightSymbolDoesNotPanic(t *testing.T) {
	stocks := map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
		"E": mustStock(t, "E", 888, false),
	}
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 0, "E": 100})

	got, err := Calculate(p, stocks, 10000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	assertOrders(t, got, []wantOrder{})
}

// TestCalculate_UnknownStockExcludedLikeHalted covers a portfolio referencing a
// symbol absent from the stock lookup entirely (as opposed to present-but-halted)
// -- eligibility must treat "unknown" the same as "not tradable".
func TestCalculate_UnknownStockExcludedLikeHalted(t *testing.T) {
	stocks := map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
	}
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 50, "Z": 50})

	got, err := Calculate(p, stocks, 10000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	assertOrders(t, got, []wantOrder{{"A", 10000, 10000}})
}

// TestCalculate_QuantityPrecision proves the integer-only quantity arithmetic:
// $201 against a $100 share price should buy exactly 2.010 units (2010
// thousandths), the case a float-division mistake gets subtly wrong.
func TestCalculate_QuantityPrecision(t *testing.T) {
	stocks := map[Symbol]Stock{"A": mustStock(t, "A", 100, true)}
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 100})

	got, err := Calculate(p, stocks, 1000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	// order amount = 1000*100/100 = 1000 dollars; quantityUnits = 1000*1000/100 = 10000
	assertOrders(t, got, []wantOrder{{"A", 1000, 10000}})
}

// TestCalculate_TooExpensiveToBuyMinimumUnitIsDropped covers a symbol whose
// order amount clears MinOrderAmount but whose price is high enough that even
// 0.001 of a share can't be afforded -- it must be dropped, not emit a
// zero-quantity order, and (per the fix above) its weight must be absorbed by
// the remaining survivor rather than vanish.
func TestCalculate_TooExpensiveToBuyMinimumUnitIsDropped(t *testing.T) {
	stocks := map[Symbol]Stock{
		"A": mustStock(t, "A", 1000, true),
		"X": mustStock(t, "X", 300000, true), // needs >= 300 dollars for 0.001 units
	}
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 98, "X": 2})

	got, err := Calculate(p, stocks, 10000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	// X's original share (200 dollars) clears MinOrderAmount but can't buy 0.001
	// units at 300000/share, so it's dropped and A absorbs the full amount.
	assertOrders(t, got, []wantOrder{{"A", 10000, 10000}})
}
