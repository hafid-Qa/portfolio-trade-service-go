package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/repositories/memory"

	"github.com/gin-gonic/gin"
)

// logoutTestRouter mirrors the real wiring: logout and a protected route
// behind the same authMiddleware and denylist.
func logoutTestRouter(t *testing.T) (*gin.Engine, *Server) {
	t.Helper()
	server := &Server{tokenMaker: newTestTokenMaker(t), denylist: memory.NewTokenDenylist()}
	router := gin.New()
	authed := router.Group("", authMiddleware(server.tokenMaker, server.denylist))
	authed.POST("/logout", server.LogoutHandler)
	authed.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router, server
}

func requestWithToken(router *gin.Engine, method, path, tok string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set(authorizationHeaderKey, "Bearer "+tok)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestLogoutHandler_RevokesOnlyThatToken(t *testing.T) {
	router, server := logoutTestRouter(t)
	loggedOut := mustToken(t, server.tokenMaker, time.Minute)
	otherDevice := mustToken(t, server.tokenMaker, time.Minute) // same user, separate login

	if w := requestWithToken(router, http.MethodGet, "/protected", loggedOut); w.Code != http.StatusOK {
		t.Fatalf("before logout: status = %d, want 200", w.Code)
	}

	if w := requestWithToken(router, http.MethodPost, "/logout", loggedOut); w.Code != http.StatusNoContent {
		t.Fatalf("logout: status = %d, want 204; body = %s", w.Code, w.Body)
	}

	if w := requestWithToken(router, http.MethodGet, "/protected", loggedOut); w.Code != http.StatusUnauthorized {
		t.Errorf("after logout: status = %d, want 401", w.Code)
	}
	if w := requestWithToken(router, http.MethodPost, "/logout", loggedOut); w.Code != http.StatusUnauthorized {
		t.Errorf("second logout with same token: status = %d, want 401", w.Code)
	}
	if w := requestWithToken(router, http.MethodGet, "/protected", otherDevice); w.Code != http.StatusOK {
		t.Errorf("other token after logout: status = %d, want 200 (only the used token is revoked)", w.Code)
	}
}

func TestLogoutHandler_RequiresAuth(t *testing.T) {
	router, _ := logoutTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
