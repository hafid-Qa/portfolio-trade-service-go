package domain

import (
	"fmt"
	"strings"
)

type Symbol string

func NewSymbol(s string) (Symbol, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("symbol cannot be empty")
	}
	return Symbol(s), nil
}

func sumTo100Percent(v map[Symbol]int) (bool, int) {
	total := 0
	for _, value := range v {
		total += value
	}
	return total == 100, total
}
