package api

import "app/internal/domain"

type StockResponse struct {
	Symbol string `json:"symbol"`
	Price  int    `json:"price"`
	// Tradable is false for halted stocks: they can sit in a portfolio, but
	// trades skip them and redistribute their share.
	Tradable bool `json:"tradable"`
}

func toStockResponses(stocks []domain.Stock) []StockResponse {
	out := make([]StockResponse, len(stocks))
	for i, s := range stocks {
		out[i] = StockResponse{Symbol: s.Symbol().String(), Price: s.Price(), Tradable: s.Tradable()}
	}
	return out
}
