package domain

import (
	"errors"
	"slices"
	"testing"
)

func TestNewPortfolio_Valid(t *testing.T) {
	p, err := NewPortfolio(1, map[Symbol]int{"A": 40, "B": 60})
	if err != nil {
		t.Fatalf("NewPortfolio() error = %v", err)
	}
	if p.UserId() != 1 {
		t.Errorf("UserId() = %d, want 1", p.UserId())
	}
}

func TestNewPortfolio_RejectsNonPositiveUserID(t *testing.T) {
	for _, userID := range []int64{0, -1} {
		_, err := NewPortfolio(userID, map[Symbol]int{"A": 100})
		if !errors.Is(err, ErrInvalidUserId) {
			t.Errorf("NewPortfolio(userID=%d) error = %v, want ErrInvalidUserId", userID, err)
		}
	}
}

func TestNewPortfolio_RejectsWeightsNotSummingTo100(t *testing.T) {
	tests := []struct {
		name    string
		weights map[Symbol]int
	}{
		{"under 100", map[Symbol]int{"A": 40, "B": 50}},
		{"over 100", map[Symbol]int{"A": 60, "B": 60}},
		{"empty portfolio", map[Symbol]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPortfolio(1, tt.weights)
			if !errors.Is(err, ErrInvalidPortfolioSum) {
				t.Errorf("NewPortfolio(%v) error = %v, want ErrInvalidPortfolioSum", tt.weights, err)
			}
		})
	}
}

// TestNewPortfolio_CopiesInputMap guards against caller-side aliasing: mutating
// the map passed into NewPortfolio after construction must not affect the
// Portfolio's internal state, since Go maps are reference types.
func TestNewPortfolio_CopiesInputMap(t *testing.T) {
	weights := map[Symbol]int{"A": 40, "B": 60}
	p := mustPortfolio(t, 1, weights)

	weights["A"] = 999

	got := p.TargetPortfolio()
	if got["A"] != 40 {
		t.Errorf("Portfolio's internal weight for A = %d, want 40 (unaffected by caller's later mutation)", got["A"])
	}
}

// TestPortfolio_TargetPortfolioReturnsACopy guards the other direction: mutating
// the map returned by TargetPortfolio() must not affect the Portfolio itself.
func TestPortfolio_TargetPortfolioReturnsACopy(t *testing.T) {
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 40, "B": 60})

	got := p.TargetPortfolio()
	got["A"] = 999

	again := p.TargetPortfolio()
	if again["A"] != 40 {
		t.Errorf("Portfolio's internal weight for A = %d after mutating a returned copy, want 40", again["A"])
	}
}

func TestPortfolio_Tickers(t *testing.T) {
	p := mustPortfolio(t, 1, map[Symbol]int{"A": 40, "B": 60})

	got := p.Tickers()
	slices.Sort(got)

	want := []Symbol{"A", "B"}
	if !slices.Equal(got, want) {
		t.Errorf("Tickers() = %v, want %v", got, want)
	}
}

func TestSumTo100Percent(t *testing.T) {
	tests := []struct {
		name    string
		weights map[Symbol]int
		wantOK  bool
		wantSum int
	}{
		{"sums to exactly 100", map[Symbol]int{"A": 40, "B": 60}, true, 100},
		{"single symbol at 100", map[Symbol]int{"A": 100}, true, 100},
		{"under 100", map[Symbol]int{"A": 40, "B": 50}, false, 90},
		{"over 100", map[Symbol]int{"A": 60, "B": 60}, false, 120},
		{"empty", map[Symbol]int{}, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, sum := sumTo100Percent(tt.weights)
			if ok != tt.wantOK || sum != tt.wantSum {
				t.Errorf("sumTo100Percent(%v) = (%v, %d), want (%v, %d)", tt.weights, ok, sum, tt.wantOK, tt.wantSum)
			}
		})
	}
}
