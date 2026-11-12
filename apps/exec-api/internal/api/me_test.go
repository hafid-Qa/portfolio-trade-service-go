package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"app/internal/domain"
	"app/internal/password"
	"app/internal/repositories/memory"

	"github.com/gin-gonic/gin"
)

func meTestRouter(t *testing.T, repo *loginUserRepo) (*gin.Engine, *Server) {
	t.Helper()
	server := &Server{
		userService: domain.NewUserService(repo, password.Argon2Hasher{}),
		tokenMaker:  newTestTokenMaker(t),
		denylist:    memory.NewTokenDenylist(),
	}
	router := gin.New()
	router.GET("/me", authMiddleware(server.tokenMaker, server.denylist), server.GetMeHandler)
	return router, server
}

func tokenFor(t *testing.T, server *Server, userID int64) string {
	t.Helper()
	tok, err := server.tokenMaker.CreateToken(userID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestGetMeHandler_ReturnsTokenOwner(t *testing.T) {
	repo := seededLoginRepo(t) // user 7, alice
	router, server := meTestRouter(t, repo)

	w := requestWithToken(router, http.MethodGet, "/me", tokenFor(t, server, 7))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}

	var resp UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if resp.ID != 7 || resp.Username != "alice" || resp.Email != "alice@example.com" {
		t.Errorf("response = %+v, want user 7 alice", resp)
	}
	if strings.Contains(w.Body.String(), repo.user.PasswordHash()) || strings.Contains(w.Body.String(), "password") {
		t.Errorf("response leaks the password hash: %s", w.Body)
	}
}

func TestGetMeHandler_RequiresAuth(t *testing.T) {
	router, _ := meTestRouter(t, seededLoginRepo(t))

	w := requestWithToken(router, http.MethodGet, "/me", "not-a-token")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// A token stays valid after its user is deleted; /me must not 500 or leak that.
func TestGetMeHandler_DeletedUserIs401(t *testing.T) {
	router, server := meTestRouter(t, seededLoginRepo(t))

	w := requestWithToken(router, http.MethodGet, "/me", tokenFor(t, server, 999))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"unauthorized"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestGetMeHandler_RepoFailureIs500(t *testing.T) {
	repo := seededLoginRepo(t)
	repo.getErr = errors.New("connection refused")
	router, server := meTestRouter(t, repo)

	w := requestWithToken(router, http.MethodGet, "/me", tokenFor(t, server, 7))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"internal server error"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}
