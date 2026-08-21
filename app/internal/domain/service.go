package domain

type TradeService struct {
	stockRepo     StockRepository
	PortfolioRepo PortfolioRepository
}

func NewTradeService(s StockRepository, p PortfolioRepository) *TradeService {
	return &TradeService{stockRepo: s, PortfolioRepo: p}
}

func (s *TradeService) CreateTrade(userID int64, amount int) (TradeResult, error) {
	portfolio, err := s.PortfolioRepo.Get(userID)
	if err != nil {
		return TradeResult{}, err
	}
	tickers := portfolio.Tickers()
	stocks, err := s.stockRepo.GetBySymbols(tickers)
	if err != nil {
		return TradeResult{}, err
	}
	if len(stocks) < len(tickers) {
		missing := []string{}
		for _, sym := range tickers {
			if _, ok := stocks[sym]; !ok {
				missing = append(missing, sym.String())
			}
		}
		return TradeResult{}, UnknownStocksInPortfolio{tickers: missing}
	}
	orders, err := Calculate(portfolio, stocks, amount)
	if err != nil {
		return TradeResult{}, err
	}
	return NewTradeResult(amount, portfolio.TargetPortfolio(), orders), nil
}
