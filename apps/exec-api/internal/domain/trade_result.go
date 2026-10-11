package domain

import (
	"maps"
	"slices"
	"time"
)

type TradeResult struct {
	id              int64
	amount          int
	targetPortfolio map[Symbol]int // map of stock symbols to their percentage allocation in the portfolio
	orders          []Order
	userID          int64
	status          TradeStatus
	createdAt       time.Time
	updatedAt       time.Time
}

func (tr TradeResult) ID() int64                       { return tr.id }
func (tr TradeResult) Amount() int                     { return tr.amount }
func (tr TradeResult) TargetPortfolio() map[Symbol]int { return maps.Clone(tr.targetPortfolio) }
func (tr TradeResult) Orders() []Order                 { return slices.Clone(tr.orders) }
func (tr TradeResult) UserID() int64 {
	return tr.userID
}
func (tr TradeResult) Status() TradeStatus {
	return tr.status
}
func (tr TradeResult) CreatedAt() time.Time {
	return tr.createdAt
}
func (tr TradeResult) UpdatedAt() time.Time {
	return tr.updatedAt
}

func NewTradeResult(id int64, amout int, targetPortfolio map[Symbol]int, orders []Order, userId int64, status TradeStatus, createdAt time.Time, updatedAt time.Time) TradeResult {
	return TradeResult{
		id:              id,
		amount:          amout,
		targetPortfolio: maps.Clone(targetPortfolio),
		orders:          slices.Clone(orders),
		userID:          userId,
		status:          status,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}
