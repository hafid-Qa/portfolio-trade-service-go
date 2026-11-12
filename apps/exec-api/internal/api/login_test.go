package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/config"
	"app/internal/domain"
	"app/internal/password"

	"github.com/gin-gonic/gin"
)

// loginUserRepo holds at most one user; getErr simulates an infrastructure failure.
type loginUserRepo struct {
	user   *domain.UserResult
	getErr error
}

func (r *loginUserRepo) GetByUsername(ctx context.Context, username string) (domain.UserResult, error) {
	if r.getErr != nil {
		return domain.UserResult{}, r.getErr
	}
	if r.user == nil || r.user.UserName() != username {
		return domain.UserResult{}, fmt.Errorf("%w: %q", domain.ErrUserNotFound, username)
	}
	return *r.user, nil
}

func (r *loginUserRepo) UpdateLastLogin(ctx context.Context, id int64) (domain.UserResult, error) {
	return *r.user, nil
}

func (r *loginUserRepo) Get(ctx context.Context, id int64) (domain.UserResult, error) {
	if r.getErr != nil {
		return domain.UserResult{}, r.getErr
	}
	if r.user == nil || r.user.ID() != id {
		return domain.UserResult{}, fmt.Errorf("%w: id %d", domain.ErrUserNotFound, id)
	}
	return *r.user, nil
}
func (r *loginUserRepo) GetByEmail(context.Context, string) (domain.UserResult, error) {
	return domain.UserResult{}, errors.New("not used")
}
func (r *loginUserRepo) Create(context.Context, string, string, string, string) (domain.UserResult, error) {
	return domain.UserResult{}, errors.New("not used")
}
func (r *loginUserRepo) UpdateProfile(context.Context, int64, *string, *string, *string) (domain.UserResult, error) {
	return domain.UserResult{}, errors.New("not used")
}
func (r *loginUserRepo) UpdatePassword(context.Context, int64, string) (domain.UserResult, error) {
	return domain.UserResult{}, errors.New("not used")
}

func loginTestRouter(t *testing.T, repo *loginUserRepo) (*gin.Engine, *Server) {
	t.Helper()
	server := &Server{
		config:      &config.Config{AccessTokenDuration: time.Minute},
		userService: domain.NewUserService(repo, password.Argon2Hasher{}),
		tokenMaker:  newTestTokenMaker(t),
	}
	router := gin.New()
	router.POST("/login", server.LoginHandler)
	return router, server
}

func seededLoginRepo(t *testing.T) *loginUserRepo {
	t.Helper()
	hash, err := password.Hash("s3cret!")
	if err != nil {
		t.Fatal(err)
	}
	u := domain.NewUserResult(7, "Alice", "alice", "alice@example.com", hash, nil, time.Now(), time.Now())
	return &loginUserRepo{user: &u}
}

func postLogin(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestLoginHandler_ReturnsUsableToken(t *testing.T) {
	router, server := loginTestRouter(t, seededLoginRepo(t))

	w := postLogin(router, `{"username":"alice","password":"s3cret!"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}

	var resp loginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if resp.Type != "Bearer" {
		t.Errorf("type = %q, want Bearer", resp.Type)
	}

	// The token must be accepted by the same maker the auth middleware uses,
	// and carry the logged-in user's ID.
	payload, err := server.tokenMaker.VerifyToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("VerifyToken(access_token) = %v", err)
	}
	if payload.UserID != 7 {
		t.Errorf("token user_id = %d, want 7", payload.UserID)
	}
}

func TestLoginHandler_GenericFailure(t *testing.T) {
	router, _ := loginTestRouter(t, seededLoginRepo(t))

	cases := map[string]string{
		"wrong password":   `{"username":"alice","password":"wrong-password"}`,
		"unknown user":     `{"username":"mallory","password":"s3cret!"}`,
		"missing password": `{"username":"alice"}`,
		"malformed json":   `{"username":`,
		"fails validation": `{"username":"a!ice","password":"x"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			w := postLogin(router, body)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
			if got, want := w.Body.String(), `{"error":"invalid credentials"}`; got != want {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}

func TestLoginHandler_RepoFailureIs500(t *testing.T) {
	repo := seededLoginRepo(t)
	repo.getErr = errors.New("connection refused")
	router, _ := loginTestRouter(t, repo)

	w := postLogin(router, `{"username":"alice","password":"s3cret!"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"internal server error"}`; got != want {
		t.Errorf("body = %s, want %s (no internal detail)", got, want)
	}
}
