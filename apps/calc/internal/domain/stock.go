package domain

type Stock struct {
	symbol   string
	price    int
	tradable bool
}

func (s Stock) Symbol() string { return s.symbol }
func (s Stock) Price() int     { return s.price }
func (s Stock) Tradable() bool { return s.tradable }
