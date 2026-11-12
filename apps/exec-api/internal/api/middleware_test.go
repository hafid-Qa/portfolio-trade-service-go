package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/repositories/memory"
	"app/internal/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const testTokenKeyHex = "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"

// authTestRouter mounts authMiddleware in front of a handler that echoes the
// authenticated user ID, so tests can assert both rejection and pass-through.
func authTestRouter(t *testing.T, maker token.Maker) *gin.Engine {
	t.Helper()
	return authTestRouterWithDenylist(t, maker, memory.NewTokenDenylist())
}

func authTestRouterWithDenylist(t *testing.T, maker token.Maker, denylist token.Denylist) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.GET("/auth", authMiddleware(maker, denylist), func(c *gin.Context) {
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

// failingDenylist simulates Redis being unreachable.
type failingDenylist struct{}

func (failingDenylist) Revoke(context.Context, uuid.UUID, time.Time) error {
	return errors.New("redis down")
}
func (failingDenylist) IsRevoked(context.Context, uuid.UUID) (bool, error) {
	return false, errors.New("redis down")
}

func TestAuthMiddleware_RevokedToken(t *testing.T) {
	maker := newTestTokenMaker(t)
	denylist := memory.NewTokenDenylist()
	router := authTestRouterWithDenylist(t, maker, denylist)
	tok := mustToken(t, maker, time.Minute)

	payload, err := maker.VerifyToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if err := denylist.Revoke(context.Background(), payload.ID, payload.ExpiredAt); err != nil {
		t.Fatal(err)
	}

	w := doAuthRequest(router, "Bearer "+tok)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	// Same body as every other auth failure: a logged-out token isn't distinguishable.
	if got, want := w.Body.String(), `{"error":"unauthorized"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

// If the denylist can't be checked, the request must be refused, not let through.
func TestAuthMiddleware_DenylistUnavailableFailsClosed(t *testing.T) {
	maker := newTestTokenMaker(t)
	router := authTestRouterWithDenylist(t, maker, failingDenylist{})

	w := doAuthRequest(router, "Bearer "+mustToken(t, maker, time.Minute))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
