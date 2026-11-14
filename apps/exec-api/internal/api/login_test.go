package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"app/config"
	"app/internal/domain"
	"app/internal/password"
	"app/internal/repositories/memory"

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
		config:       &config.Config{AccessTokenDuration: time.Minute},
		userService:  domain.NewUserService(repo, password.Argon2Hasher{}),
		tokenMaker:   newTestTokenMaker(t),
		loginLimiter: memory.NewLoginLimiter(testLoginMaxAttempts, testLoginPeriod),
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
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
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

// postLoginForm sends the request the way Swagger UI's OAuth2 password flow does.
func postLoginForm(router *gin.Engine, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestLoginHandler_OAuth2PasswordForm(t *testing.T) {
	router, server := loginTestRouter(t, seededLoginRepo(t))

	w := postLoginForm(router, url.Values{
		"grant_type": {"password"},
		"username":   {"alice"},
		"password":   {"s3cret!"},
		"client_id":  {""},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}

	// Swagger UI reads exactly these two OAuth2 fields.
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if resp["token_type"] != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp["token_type"])
	}
	payload, err := server.tokenMaker.VerifyToken(resp["access_token"])
	if err != nil {
		t.Fatalf("VerifyToken(access_token) = %v", err)
	}
	if payload.UserID != 7 {
		t.Errorf("token user_id = %d, want 7", payload.UserID)
	}
}

func TestLoginHandler_OAuth2PasswordForm_WrongPassword(t *testing.T) {
	router, _ := loginTestRouter(t, seededLoginRepo(t))

	w := postLoginForm(router, url.Values{
		"grant_type": {"password"},
		"username":   {"alice"},
		"password":   {"wrong-password"},
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"invalid credentials"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

// --- login rate limiting

// The limits the test servers use (production values come from config).
const (
	testLoginMaxAttempts = 5
	testLoginPeriod      = 15 * time.Minute
)

// failingLimiter simulates Redis being unreachable.
type failingLimiter struct{}

func (failingLimiter) Allow(context.Context, string) (bool, time.Duration, error) {
	return false, 0, errors.New("redis down")
}
func (failingLimiter) Reset(context.Context, string) error { return nil }

func TestLoginHandler_RateLimitedAfterMaxFailures(t *testing.T) {
	router, _ := loginTestRouter(t, seededLoginRepo(t))
	wrong := `{"username":"alice","password":"wrong-password"}`

	for i := range testLoginMaxAttempts {
		if w := postLogin(router, wrong); w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, w.Code)
		}
	}

	// Over the limit: refused even with the CORRECT password.
	w := postLogin(router, `{"username":"alice","password":"s3cret!"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"too many login attempts, try again later"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	retry, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil || retry <= 0 || retry > int(testLoginPeriod.Seconds()) {
		t.Errorf("Retry-After = %q, want seconds in (0, %d]", w.Header().Get("Retry-After"), int(testLoginPeriod.Seconds()))
	}
}

// A successful login resets the count, so only consecutive failures add up.
func TestLoginHandler_SuccessResetsAttempts(t *testing.T) {
	router, _ := loginTestRouter(t, seededLoginRepo(t))
	wrong := `{"username":"alice","password":"wrong-password"}`

	for range testLoginMaxAttempts - 1 {
		postLogin(router, wrong)
	}
	if w := postLogin(router, `{"username":"alice","password":"s3cret!"}`); w.Code != http.StatusOK {
		t.Fatalf("login before limit: status = %d, want 200", w.Code)
	}
	for i := range testLoginMaxAttempts - 1 {
		if w := postLogin(router, wrong); w.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d after reset: status = %d, want 401 (count should restart)", i+1, w.Code)
		}
	}
}

// Unknown usernames are limited exactly like real ones, so the 429 can't be
// used to tell which accounts exist; and the key ignores case.
func TestLoginHandler_RateLimitIsPerUsername(t *testing.T) {
	router, _ := loginTestRouter(t, seededLoginRepo(t))

	for range testLoginMaxAttempts {
		postLogin(router, `{"username":"mallory","password":"whatever1"}`)
	}
	if w := postLogin(router, `{"username":"MALLORY","password":"whatever1"}`); w.Code != http.StatusTooManyRequests {
		t.Errorf("unknown user over limit (different case): status = %d, want 429", w.Code)
	}
	// Another username is unaffected.
	if w := postLogin(router, `{"username":"alice","password":"s3cret!"}`); w.Code != http.StatusOK {
		t.Errorf("other user: status = %d, want 200", w.Code)
	}
}

// If the limiter can't be checked, login is refused (fail closed).
func TestLoginHandler_LimiterUnavailableIs500(t *testing.T) {
	router, server := loginTestRouter(t, seededLoginRepo(t))
	server.loginLimiter = failingLimiter{}

	w := postLogin(router, `{"username":"alice","password":"s3cret!"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
