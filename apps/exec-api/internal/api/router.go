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
// @Tags Trades
// @Accept json
// @Produce json
// @Security OAuth2Password
// @Param request body TradeRequest true "Trade amount"
// @Success 200 {object} TradeResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/trades [post]
func (server *Server) TradeHandler(c *gin.Context) {
	userID := authPayload(c).UserID

	var req TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	res, err := server.tradeService.CreateTrade(c.Request.Context(), userID, req.Amount)
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

// @Tags Trades
// @Accept json
// @Produce json
// @Security OAuth2Password
// @Param id path int true "Trade ID"
// @Success 200 {object} TradeResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/trades/{id} [get]
func (server *Server) GetTradeHanlder(c *gin.Context) {
	var uri GetTradeRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	res, err := server.tradeService.GetTrade(c.Request.Context(), authPayload(c).UserID, uri.ID)
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

// @Summary List a user's trades
// @Description Returns a paginated list of trades for a user, most recent first.
// @Tags Trades
// @Accept json
// @Produce json
// @Security OAuth2Password
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(50)
// @Success 200 {object} PaginatedResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/trades [get]
func (server *Server) GetTradeByUserHanlder(c *gin.Context) {
	var query Pagination
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	trades, totalCount, err := server.tradeService.GetTradesByUserID(c.Request.Context(), authPayload(c).UserID, query.Page, query.Limit)
	if err != nil {

		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	tradeRes := make([]TradeResponse, len(trades))
	for idx, res := range trades {
		tradeRes[idx] = toTradeResponse(res)
	}
	paginatedRes := PaginatedResponse{
		Data:       tradeRes,
		TotalCount: int(totalCount),
		Page:       int(query.Page),
		Limit:      int(query.Limit),
	}

	c.JSON(http.StatusOK, paginatedRes)
}

// @Summary Create a user
// @Description Registers a new user. Username and email are stored lowercased
// @Description and must be unique.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UserRequest true "New user"
// @Success 201 {object} UserResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users [post]
func (server *Server) CreateUserHandler(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.userService.Register(c.Request.Context(), req.FullName, req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			// Deliberately doesn't say which field clashed.
			c.JSON(http.StatusConflict, errorResponse(errUserAlreadyExists))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(errInternal))
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// @Summary Current user
// @Description Returns the profile of the user the access token belongs to.
// @Tags Users
// @Produce json
// @Security OAuth2Password
// @Success 200 {object} UserResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users/me [get]
func (server *Server) GetMeHandler(c *gin.Context) {
	user, err := server.userService.GetUser(c.Request.Context(), authPayload(c).UserID)
	if err != nil {
		// A valid token for a user that no longer exists (deleted since login)
		// is treated like any other bad token.
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, errorResponse(errUnauthorized))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(errInternal))
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

var (
	errUserAlreadyExists = errors.New("username or email already taken")

	// errInvalidCredentials is the only error login reports to the client:
	// a malformed request, unknown user and wrong password all look the same.
	errInvalidCredentials = errors.New("invalid credentials")
	errInternal           = errors.New("internal server error")
)

// @Summary Log in
// @Description Exchanges username and password for an access token. Every
// @Description credential failure returns the same 401.
// @Tags Users
// @Accept json,x-www-form-urlencoded
// @Produce json
// @Param request body loginUserRequest true "Credentials"
// @Success 200 {object} loginResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users/login [post]
func (server *Server) LoginHandler(c *gin.Context) {
	var req loginUserRequest
	// ShouldBind picks JSON or form binding from the Content-Type.
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse(errInvalidCredentials))
		return
	}

	user, err := server.userService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errorResponse(errInvalidCredentials))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(errInternal))
		return
	}

	accessToken, err := server.tokenMaker.CreateToken(user.ID(), server.config.AccessTokenDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(errInternal))
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	})
}

// @Summary Log out
// @Description Revokes the access token used for this request; it is rejected
// @Description from then on, even before it expires.
// @Tags Users
// @Security OAuth2Password
// @Success 204
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/users/logout [post]
func (server *Server) LogoutHandler(c *gin.Context) {
	payload := authPayload(c)
	if err := server.denylist.Revoke(c.Request.Context(), payload.ID, payload.ExpiredAt); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(errInternal))
		return
	}
	c.Status(http.StatusNoContent)
}
