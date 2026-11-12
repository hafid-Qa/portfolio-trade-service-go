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

// errUnauthorized is the only error the middleware returns for a bad token:
// missing header, bad format, wrong scheme, invalid/expired and logged-out
// tokens all look the same to the client.
var errUnauthorized = errors.New("unauthorized")

func authMiddleware(tokenMaker token.Maker, denylist token.Denylist) gin.HandlerFunc {
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

		revoked, err := denylist.IsRevoked(c.Request.Context(), payload.ID)
		if err != nil {
			// Fail closed: if we can't tell whether the token was logged out,
			// don't let it through.
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(errInternal))
			return
		}
		if revoked {
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
