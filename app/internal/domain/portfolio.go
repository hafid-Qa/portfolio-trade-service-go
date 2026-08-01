package domain

import "fmt"

type Portfolio struct {
	userId          int
	targetPortfolio map[Symbol]int // map of stock symbols to their percentage allocation in the portfolio
}

func NewPortfolio(userId int, targetPortfolio map[Symbol]int) (Portfolio, error) {

	//  UserId positive int
	if userId <= 0 {
		return Portfolio{}, fmt.Errorf("%w: user id %d", ErrInvalidUserId, userId)
	}
	if !sumTo100Percent(targetPortfolio) {
		return Portfolio{}, fmt.Errorf("%w", ErrInvalidPortfolioSum)
	}
	return Portfolio{
		userId:          userId,
		targetPortfolio: targetPortfolio,
	}, nil

}
