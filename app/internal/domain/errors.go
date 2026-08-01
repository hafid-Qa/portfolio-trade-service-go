package domain

import (
	"errors"
	"fmt"
	"strings"
)

type PortfolioNotFound struct {
	userId int
}

func (e PortfolioNotFound) Error() string {
	return fmt.Sprintf("No portfolio found for user %d", e.userId)

}

type UnknownStocksInPortfolio struct {
	tickers []string
}

func (e UnknownStocksInPortfolio) Error() string {
	tickers := strings.Join(e.tickers, ", ")
	return fmt.Sprintf("Stocks not found in catalogue: %s", tickers)

}

type TradeAmountBelowMinimum struct {
	amount   int
	minTrade int
}

func (e TradeAmountBelowMinimum) Error() string {

	return fmt.Sprintf("Trade amount %d is below the minimum of %d", e.amount, e.minTrade)

}

var (
	ErrInvalidStock      = errors.New("invalid stock")
	ErrInvalidPortfolio  = errors.New("invalid portfolio")
	ErrPortfolioNotFound = errors.New("portfolio not found")
	ErrUnknownStock      = errors.New("portfolio references unknown stock")
	ErrInvalidSymbol     = errors.New("invalid symbol")
)
