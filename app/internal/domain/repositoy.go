package domain

type StockRepository interface {
	All() (map[Symbol]Stock, error)
}

type PortfolioRespository interface {
	Get(UserId int) (Portfolio, error)
}
