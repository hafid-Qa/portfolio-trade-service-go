package domain

import "fmt"

type Stock struct {
	ticker   Ticker
	price    int
	tradable bool
}

func (s Stock) Ticker() Ticker { return s.ticker }
func (s Stock) Price() int     { return s.price }
func (s Stock) Tradable() bool { return s.tradable }
func NewStock(ticker string, price int, tradable bool) (Stock, error) {

	s, err := NewSymbol(ticker)
	if err != nil {
		return Stock{}, fmt.Errorf("%w: %w", ErrInvalidSymbol, err)
	}
	if price <= 0 {
		return Stock{}, fmt.Errorf("%w: price must be positive, got %d", ErrInvalidStock, price)
	}
	return Stock{
		ticker:   s,
		price:    price,
		tradable: tradable,
	}, nil

}
