package api

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/token"

	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// errUnauthorized is the only error the middleware returns: missing header,
// bad format, wrong scheme and invalid/expired token all look the same to
// the client.
var errUnauthorized = errors.New("unauthorized")

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		fields := strings.Fields(c.GetHeader(authorizationHeaderKey))
		if len(fields) != 2 || strings.ToLower(fields[0]) != authorizationTypeBearer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errUnauthorized))
			return
		}

		payload, err := tokenMaker.VerifyToken(fields[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errUnauthorized))
			return
		}

		c.Set(authorizationPayloadKey, payload)
		c.Next()
	}
}

// authPayload returns the verified token payload set by authMiddleware.
// It panics (500 via gin's recovery) only if a route is missing the
// middleware, which is a programming error rather than a client error.
func authPayload(c *gin.Context) *token.Payload {
	return c.MustGet(authorizationPayloadKey).(*token.Payload)
}
