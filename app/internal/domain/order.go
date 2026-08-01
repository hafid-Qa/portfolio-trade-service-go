package domain

type Order struct {
	symbol   Symbol
	amount   int
	quantity float64
}

func (o Order) Symbol() Symbol { return o.symbol }
func (o Order) Amount() int          { return o.amount }
func (o Order) Quantity() float64    { return o.quantity }

func NewOrder(symbol Symbol, amount int, quantity float64) Order {
	return Order{
		symbol:   symbol,
		amount:   amount,
		quantity: quantity,
	}
}
