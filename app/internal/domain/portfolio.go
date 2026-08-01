package domain

import "fmt"

type Portfolio struct {
	userId          int
	targetPortfolio map[Symbol]int // map of stock symbols to their percentage allocation in the portfolio
}

func (p Portfolio) UserId() int { return p.userId }

func (p Portfolio) TargetPortfolio() map[Symbol]int {
	return copyPortfolio(p.targetPortfolio)
}

func NewPortfolio(userId int, targetPortfolio map[Symbol]int) (Portfolio, error) {
	if userId <= 0 {
		return Portfolio{}, fmt.Errorf("%w: user id %d", ErrInvalidUserId, userId)
	}

	ok, sum := sumTo100Percent(targetPortfolio)
	if !ok {
		return Portfolio{}, fmt.Errorf("%w: got %d%%", ErrInvalidPortfolioSum, sum)
	}

	return Portfolio{
		userId:          userId,
		targetPortfolio: copyPortfolio(targetPortfolio),
	}, nil
}

func copyPortfolio(v map[Symbol]int) map[Symbol]int {
	copied := make(map[Symbol]int, len(v))
	for symbol, weight := range v {
		copied[symbol] = weight
	}
	return copied
}
