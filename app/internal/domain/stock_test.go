package domain

import (
	"errors"
	"testing"
)

func TestNewStock_Valid(t *testing.T) {
	s, err := NewStock("A", 1000, true)
	if err != nil {
		t.Fatalf("NewStock() error = %v", err)
	}
	if s.Symbol() != "A" || s.Price() != 1000 || !s.Tradable() {
		t.Errorf("NewStock() = %+v, want {A 1000 true}", s)
	}
}

func TestNewStock_RejectsEmptyTicker(t *testing.T) {
	_, err := NewStock("", 1000, true)
	if !errors.Is(err, ErrInvalidStock) {
		t.Errorf("NewStock(empty ticker) error = %v, want ErrInvalidStock", err)
	}
}

func TestNewStock_RejectsNonPositivePrice(t *testing.T) {
	for _, price := range []int{0, -1} {
		_, err := NewStock("A", price, true)
		if !errors.Is(err, ErrInvalidStock) {
			t.Errorf("NewStock(price=%d) error = %v, want ErrInvalidStock", price, err)
		}
	}
}

func TestNewSymbol(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Symbol
		wantErr bool
	}{
		{"plain ticker", "A", "A", false},
		{"trims surrounding whitespace", "  A  ", "A", false},
		{"empty string", "", "", true},
		{"whitespace only", "   ", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSymbol(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewSymbol(%q) error = nil, want an error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewSymbol(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("NewSymbol(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSymbol_String(t *testing.T) {
	s := Symbol("A")
	if s.String() != "A" {
		t.Errorf("Symbol(\"A\").String() = %q, want \"A\"", s.String())
	}
}
