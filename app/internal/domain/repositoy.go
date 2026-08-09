package domain

type StockRepository interface {
	All() (map[Symbol]Stock, error)
	GetBySymbols(symbols []Symbol) (map[Symbol]Stock, error)
}

type PortfolioRespository interface {
	Get(UserId int) (Portfolio, error)
}
