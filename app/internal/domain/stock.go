package domain

import (
	"fmt"
	"strings"
)

type Symbol string

func NewSymbol(s string) (Symbol, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("symbol cannot be empty")
	}
	return Symbol(s), nil
}

func (s Symbol) String() string {
	return string(s)
}

type Stock struct {
	symbol   Symbol
	price    int
	tradable bool
}

func (s Stock) Symbol() Symbol { return s.symbol }
func (s Stock) Price() int     { return s.price }
func (s Stock) Tradable() bool { return s.tradable }

func NewStock(ticker string, price int, tradable bool) (Stock, error) {
	symbol, err := NewSymbol(ticker)
	if err != nil {
		return Stock{}, fmt.Errorf("%w: %w", ErrInvalidStock, err)
	}
	if price <= 0 {
		return Stock{}, fmt.Errorf("%w: price must be positive, got %d", ErrInvalidStock, price)
	}
	return Stock{
		symbol:   symbol,
		price:    price,
		tradable: tradable,
	}, nil
}
