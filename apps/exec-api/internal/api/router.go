package api

import (
	"errors"
	"net/http"

	"app/internal/domain"

	"github.com/gin-gonic/gin"
)

// toTradeResponse converts a persisted trade into its API representation --
// shared by every handler that returns one, so the OrderResponse/TradeResponse
// mapping lives in exactly one place.
func toTradeResponse(res domain.TradeResult) TradeResponse {
	orders := make([]OrderResponse, len(res.Orders()))
	for i, o := range res.Orders() {
		orders[i] = OrderResponse{
			Symbol:   o.Symbol().String(),
			Amount:   o.Amount(),
			Quantity: float64(o.QuantityUnits()) / float64(domain.QuantityPrecision),
		}
	}

	return TradeResponse{
		ID:              res.ID(),
		Amount:          res.Amount(),
		TargetPortfolio: res.TargetPortfolio(),
		Orders:          orders,
		Status:          string(res.Status()),
		CreatedAt:       res.CreatedAt(),
		UpdatedAt:       res.UpdatedAt(),
	}
}

// @Summary Health Check
// @Description Check the health of the API
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/health [get]
func (server *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}

// @Summary Create a trade for a user
// @Description Calculates buy orders for a user's target portfolio from a given trade amount.
// @Description The amount is apportioned across the portfolio's stocks by their target percentages;
// @Description stocks that are not tradable or whose allocation falls below the minimum order amount
// @Description are excluded and the remainder is redistributed among the rest.
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param request body TradeRequest true "Trade amount"
// @Success 200 {object} TradeResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users/:user_id/trades [post]
func (server *Server) TradeHandler(c *gin.Context) {
	var uri TradeRequestByUser
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	res, err := server.tradeService.CreateTrade(c.Request.Context(), uri.UserID, req.Amount)
	if err != nil {
		if errors.Is(err, domain.ErrPortfolioNotFound) {
			c.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, toTradeResponse(res))
}

// @Tags Users
// @Accept json
// @Produce json
// @Router /api/users/:user_id/trade/:id [get]
func (server *Server) GetTradeHanlder(c *gin.Context) {
	var uri GetTradeRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	res, err := server.tradeService.GetTrade(c.Request.Context(), uri.UserID, uri.ID)
	if err != nil {
		if errors.Is(err, domain.ErrTradeNotFound) {
			c.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, toTradeResponse(res))
}

// @Tags Users
// @Accept json
// @Produce json
// @Router /api/users/{user_id}/trades} [get]
func (server *Server) GetTradeByUserHanlder(c *gin.Context) {
	var uri TradeRequestByUser
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	trades, err := server.tradeService.GetTradesByUserID(c.Request.Context(), uri.UserID)
	if err != nil {

		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	tradeRes := make([]TradeResponse, len(trades))
	for idx, res := range trades {
		tradeRes[idx] = toTradeResponse(res)
	}
	c.JSON(http.StatusOK, tradeRes)
}
