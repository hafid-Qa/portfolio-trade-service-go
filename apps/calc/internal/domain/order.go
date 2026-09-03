package domain

type Order struct {
	symbol        string
	amount        int
	quantityUnits int
}

func (o Order) Symbol() string     { return o.symbol }
func (o Order) Amount() int        { return o.amount }
func (o Order) QuantityUnits() int { return o.quantityUnits }

func NewOrder(symbol string, amount int, quantityUnits int) Order {
	return Order{
		symbol:        symbol,
		amount:        amount,
		quantityUnits: quantityUnits,
	}
}
