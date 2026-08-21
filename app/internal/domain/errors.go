package domain

import (
	"errors"
	"fmt"
	"strings"
)

// A custom type (below) is for when a caller needs to extract data for the message
// (errors.As); a sentinel (the var block at the bottom) is for when it doesn't
// (errors.Is). Two different matching functions, easy to reach for the wrong one.

type PortfolioNotFound struct {
	userId int
}

func (e PortfolioNotFound) Error() string {
	return fmt.Sprintf("no portfolio found for user %d", e.userId)

}

type UnknownStocksInPortfolio struct {
	tickers []string
}

func (e UnknownStocksInPortfolio) Error() string {
	tickers := strings.Join(e.tickers, ", ")
	return fmt.Sprintf("stocks not found in catalogue: %s", tickers)

}

type TradeAmountBelowMinimum struct {
	amount   int
	minTrade int
}

func (e TradeAmountBelowMinimum) Error() string {

	return fmt.Sprintf("trade amount %d is below the minimum of %d", e.amount, e.minTrade)

}

var (
	ErrInvalidStock      = errors.New("invalid stock")
	ErrInvalidPortfolio  = errors.New("invalid portfolio")
	ErrPortfolioNotFound = errors.New("portfolio not found")
	ErrUnknownStock      = errors.New("portfolio references unknown stock")
	ErrInvalidSymbol     = errors.New("invalid symbol")
	ErrInvalidUserId     = errors.New("invalid user id")
	ErrInvalidPortfolioSum = errors.New("sum of portfolio is not 100%")
)
