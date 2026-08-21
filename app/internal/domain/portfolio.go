package domain

import (
	"fmt"
	"maps"
	"slices"
)

type Portfolio struct {
	userId          int64
	targetPortfolio map[Symbol]int // map of stock symbols to their percentage allocation in the portfolio
}

func (p Portfolio) UserId() int64 { return p.userId }

func (p Portfolio) TargetPortfolio() map[Symbol]int {
	return maps.Clone(p.targetPortfolio)
}
func (p Portfolio) Tickers() []Symbol {
	return slices.Collect(maps.Keys(p.targetPortfolio))
}

func NewPortfolio(userId int64, targetPortfolio map[Symbol]int) (Portfolio, error) {
	if userId <= 0 {
		return Portfolio{}, fmt.Errorf("%w: user id %d", ErrInvalidUserId, userId)
	}

	ok, sum := sumTo100Percent(targetPortfolio)
	if !ok {
		return Portfolio{}, fmt.Errorf("%w: got %d%%", ErrInvalidPortfolioSum, sum)
	}

	return Portfolio{
		userId:          userId,
		targetPortfolio: maps.Clone(targetPortfolio),
	}, nil
}

func sumTo100Percent(v map[Symbol]int) (bool, int) {
	total := 0
	for _, value := range v {
		total += value
	}
	return total == 100, total
}
