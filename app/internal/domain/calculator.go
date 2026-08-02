package domain

import (
	"maps"
	"slices"
)

func Calculate(p Portfolio, stocks map[Symbol]Stock, amount int) ([]Order, error) {
	if amount < MinTradeAmount {
		return []Order{}, TradeAmountBelowMinimum{
			amount:   amount,
			minTrade: MinTradeAmount,
		}
	}
	validSymbols := []Symbol{}
	targetPortfolio := p.TargetPortfolio()
	sortedSymbol := slices.Sorted(maps.Keys(targetPortfolio))
	ratioSum := 0
	for _, symbol := range sortedSymbol {
		ratio := targetPortfolio[symbol]
		stock, ok := stocks[symbol]
		if !ok || !stock.Tradable() {
			continue
		}
		stockValue := (amount * ratio) / 100
		if stockValue >= MinOrderAmount {
			validSymbols = append(validSymbols, symbol)
			ratioSum += ratio

		}
	}
	orders := []Order{}
	for _, symbol := range validSymbols {
		currentRatio := targetPortfolio[symbol]
		orderAmount := (currentRatio * amount) / ratioSum
		stockPrice := stocks[symbol].Price()
		quantityThousandths := (orderAmount * QuantityPrecision) / stockPrice
		if quantityThousandths == 0 {
			continue
		}
		quantity := float64(quantityThousandths) / float64(QuantityPrecision)
		orders = append(orders, NewOrder(symbol, orderAmount, quantity))

	}
	return orders, nil
}
