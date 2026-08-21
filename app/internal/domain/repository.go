package domain

type StockRepository interface {
	All() (map[Symbol]Stock, error)
	GetBySymbols(symbols []Symbol) (map[Symbol]Stock, error)
}

type PortfolioRepository interface {
	Get(UserId int64) (Portfolio, error)
}
