package domain

import (
	"maps"
	"slices"
)

func Calculate(portfolio map[string]int, stocks map[string]Stock, investmentAmount int, minOrderAmount int, quantityPrecision int) ([]Order, error) {

	eligibleSymbols := []string{}
	targetPortfolio := portfolio
	sortedSymbol := slices.Sorted(maps.Keys(targetPortfolio))
	for _, symbol := range sortedSymbol {
		stock, ok := stocks[symbol]
		if !ok || !stock.Tradable() || targetPortfolio[symbol] <= 0 {
			continue
		}
		eligibleSymbols = append(eligibleSymbols, symbol)
	}

	ratioSum := 0
	for {
		ratioSum = 0
		for _, t := range eligibleSymbols {
			ratioSum += targetPortfolio[t]
		}
		survivors := []string{}
		for _, t := range eligibleSymbols {
			if isOrderable(targetPortfolio[t], investmentAmount, ratioSum, stocks[t].Price(), minOrderAmount, quantityPrecision) {
				survivors = append(survivors, t)
			}
		}

		if len(eligibleSymbols) == len(survivors) {
			break
		}
		eligibleSymbols = survivors
	}

	orders := []Order{}
	if len(eligibleSymbols) == 0 {
		return orders, nil
	}
	for _, t := range eligibleSymbols {
		orderAmount := apportion(targetPortfolio[t], investmentAmount, ratioSum)
		quantity := quantityUnits(orderAmount, stocks[t].Price(), quantityPrecision)
		orders = append(orders, NewOrder(t, orderAmount, quantity))
	}
	return orders, nil
}

// Dollar amount allocated to one ticker, floored. Integer-only by construction.
func apportion(ratio int, amount int, ratioSum int) int {
	return (ratio * amount) / ratioSum
}

// Largest count of 1/QUANTITY_PRECISION-unit shares whose cost fits within order_amount.
// Stays an integer count of units all the way through the domain layer; the API
// schema is the one place that divides by QUANTITY_PRECISION to place the decimal
// point, so no float ever appears in this arithmetic.
func quantityUnits(orderAmount int, price int, quantityPrecision int) int {
	return (orderAmount * quantityPrecision) / price
}

// Whether this ticker's share produces an order that can actually be placed
func isOrderable(ratio int, amount int, ratioSum int, price int, minOrderAmount int, quantityPrecision int) bool {
	orderAmount := apportion(ratio, amount, ratioSum)
	if orderAmount < minOrderAmount {
		return false
	}
	return quantityUnits(orderAmount, price, quantityPrecision) > 0
}
