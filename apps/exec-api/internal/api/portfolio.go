package api

import "app/internal/domain"

type PortfolioRequest struct {
	// TargetPortfolio maps ticker -> percentage; weights must be 1-100 and sum to 100.
	TargetPortfolio map[string]int `json:"target_portfolio" binding:"required"`
}

type PortfolioResponse struct {
	TargetPortfolio map[string]int `json:"target_portfolio"`
}

func toPortfolioResponse(p domain.Portfolio) PortfolioResponse {
	weights := make(map[string]int, len(p.TargetPortfolio()))
	for sym, w := range p.TargetPortfolio() {
		weights[sym.String()] = w
	}
	return PortfolioResponse{TargetPortfolio: weights}
}
