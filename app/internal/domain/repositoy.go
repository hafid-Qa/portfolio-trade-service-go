package domain

type StockRepository interface {
	all() (map[Symbol]Stock, error)
}

type PortfolioRespository interface {
	Get(Portfolio, error)
}
