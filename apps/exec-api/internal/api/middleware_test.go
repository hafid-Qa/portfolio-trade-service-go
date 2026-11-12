package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/token"

	"github.com/gin-gonic/gin"
)

const testTokenKeyHex = "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"

// authTestRouter mounts authMiddleware in front of a handler that echoes the
// authenticated user ID, so tests can assert both rejection and pass-through.
func authTestRouter(t *testing.T, maker token.Maker) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.GET("/auth", authMiddleware(maker), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": authPayload(c).UserID})
	})
	return router
}

func newTestTokenMaker(t *testing.T) token.Maker {
	t.Helper()
	maker, err := token.NewPASETOMaker(testTokenKeyHex)
	if err != nil {
		t.Fatalf("NewPASETOMaker: %v", err)
	}
	return maker
}

func mustToken(t *testing.T, maker token.Maker, d time.Duration) string {
	t.Helper()
	tok, err := maker.CreateToken(42, d)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return tok
}

func doAuthRequest(router *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	if authHeader != "" {
		req.Header.Set(authorizationHeaderKey, authHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	maker := newTestTokenMaker(t)
	router := authTestRouter(t, maker)
	tok := mustToken(t, maker, time.Minute)

	for _, scheme := range []string{"Bearer", "bearer", "BEARER"} {
		t.Run(scheme, func(t *testing.T) {
			w := doAuthRequest(router, scheme+" "+tok)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
			}
			var body struct {
				UserID int64 `json:"user_id"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body.UserID != 42 {
				t.Errorf("user_id = %d, want 42", body.UserID)
			}
		})
	}
}

func TestAuthMiddleware_Rejects(t *testing.T) {
	maker := newTestTokenMaker(t)
	router := authTestRouter(t, maker)
	tok := mustToken(t, maker, time.Minute)
	expired := mustToken(t, maker, -time.Minute)

	cases := map[string]string{
		"missing header":     "",
		"token only":         tok,
		"wrong scheme":       "Basic " + tok,
		"extra field":        "Bearer " + tok + " extra",
		"expired token":      "Bearer " + expired,
		"garbage token":      "Bearer not-a-token",
		"scheme without tok": "Bearer",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			w := doAuthRequest(router, header)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
			// Every failure must look identical to the client.
			if got, want := w.Body.String(), `{"error":"unauthorized"}`; got != want {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}
