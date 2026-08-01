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
