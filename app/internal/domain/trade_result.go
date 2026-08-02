package domain

import (
	"maps"
	"slices"
)

type TradeResult struct {
	amount          int
	targetPortfolio map[Symbol]int // map of stock symbols to their percentage allocation in the portfolio
	orders          []Order
}

func (tr TradeResult) Amount() int                     { return tr.amount }
func (tr TradeResult) TargetPortfolio() map[Symbol]int { return maps.Clone(tr.targetPortfolio) }
func (tr TradeResult) Orders() []Order                 { return slices.Clone(tr.orders) }
func NewTradeResult(amout int, targetPortfolio map[Symbol]int, orders []Order) TradeResult {
	return TradeResult{
		amount:          amout,
		targetPortfolio: maps.Clone(targetPortfolio),
		orders:          slices.Clone(orders),
	}
}
