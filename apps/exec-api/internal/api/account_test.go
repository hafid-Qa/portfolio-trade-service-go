package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/config"
	"app/internal/domain"
	"app/internal/password"
	"app/internal/repositories/memory"

	"github.com/gin-gonic/gin"
)

// accountUserRepo is loginUserRepo (one stored user) plus working
// UpdateProfile/UpdatePassword; updateErr simulates the DB rejecting a change.
type accountUserRepo struct {
	loginUserRepo
	updateErr error
}

func (r *accountUserRepo) UpdateProfile(ctx context.Context, id int64, fullName, userName, email *string) (domain.UserResult, error) {
	if r.updateErr != nil {
		return domain.UserResult{}, r.updateErr
	}
	u := *r.user
	if u.ID() != id {
		return domain.UserResult{}, fmt.Errorf("%w: id %d", domain.ErrUserNotFound, id)
	}
	name, handle, mail := u.FullName(), u.UserName(), u.Email()
	if fullName != nil {
		name = *fullName
	}
	if userName != nil {
		handle = strings.ToLower(*userName)
	}
	if email != nil {
		mail = strings.ToLower(*email)
	}
	updated := domain.NewUserResult(u.ID(), name, handle, mail, u.PasswordHash(), u.LastLogin(), u.CreatedAt(), time.Now())
	r.user = &updated
	return updated, nil
}

func (r *accountUserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) (domain.UserResult, error) {
	if r.updateErr != nil {
		return domain.UserResult{}, r.updateErr
	}
	u := *r.user
	updated := domain.NewUserResult(u.ID(), u.FullName(), u.UserName(), u.Email(), passwordHash, u.LastLogin(), u.CreatedAt(), time.Now())
	r.user = &updated
	return updated, nil
}

// accountTestRouter wires the real routes for /users/me under authMiddleware,
// plus login, so tests can check a changed password end to end.
func accountTestRouter(t *testing.T) (*gin.Engine, *Server, *accountUserRepo) {
	t.Helper()
	repo := &accountUserRepo{loginUserRepo: *seededLoginRepo(t)} // user 7 alice / "s3cret!"
	server := &Server{
		config:      &config.Config{AccessTokenDuration: time.Minute},
		userService: domain.NewUserService(repo, password.Argon2Hasher{}),
		tokenMaker:  newTestTokenMaker(t),
		denylist:    memory.NewTokenDenylist(),
	}
	router := gin.New()
	router.POST("/login", server.LoginHandler)
	authed := router.Group("", authMiddleware(server.tokenMaker, server.denylist))
	authed.PATCH("/me", server.UpdateMeHandler)
	authed.PUT("/me/password", server.ChangePasswordHandler)
	return router, server, repo
}

func sendJSONAs(t *testing.T, router *gin.Engine, server *Server, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authorizationHeaderKey, "Bearer "+tokenFor(t, server, userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// --- PATCH /users/me

func TestUpdateMeHandler_PartialUpdate(t *testing.T) {
	router, server, repo := accountTestRouter(t)

	w := sendJSONAs(t, router, server, 7, http.MethodPatch, "/me", `{"full_name":"Alice Cooper"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}
	var resp UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.FullName != "Alice Cooper" {
		t.Errorf("full_name = %q, want Alice Cooper", resp.FullName)
	}
	// Fields not sent are left alone.
	if resp.Username != "alice" || resp.Email != "alice@example.com" {
		t.Errorf("untouched fields changed: %+v", resp)
	}
	if repo.user.FullName() != "Alice Cooper" {
		t.Error("change not stored")
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Errorf("response leaks password data: %s", w.Body)
	}
}

func TestUpdateMeHandler_AllFields(t *testing.T) {
	router, server, _ := accountTestRouter(t)

	w := sendJSONAs(t, router, server, 7, http.MethodPatch, "/me",
		`{"full_name":"Al","username":"AliceC","email":"AC@Example.com"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}
	var resp UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.FullName != "Al" || resp.Username != "alicec" || resp.Email != "ac@example.com" {
		t.Errorf("response = %+v, want updated + lowercased username/email", resp)
	}
}

func TestUpdateMeHandler_Duplicate(t *testing.T) {
	router, server, repo := accountTestRouter(t)
	repo.updateErr = fmt.Errorf("%w: users_username_key", domain.ErrUserAlreadyExists)

	w := sendJSONAs(t, router, server, 7, http.MethodPatch, "/me", `{"username":"bob"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"username or email already taken"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestUpdateMeHandler_Invalid(t *testing.T) {
	cases := map[string]string{
		"no fields":          `{}`,
		"empty full_name":    `{"full_name":""}`,
		"empty username":     `{"username":""}`,
		"non-alnum username": `{"username":"al ice"}`,
		"username too long":  `{"username":"` + strings.Repeat("a", 51) + `"}`,
		"bad email":          `{"email":"not-an-email"}`,
		"full_name too long": `{"full_name":"` + strings.Repeat("a", 101) + `"}`,
		"malformed json":     `{"full_name":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			router, server, repo := accountTestRouter(t)
			w := sendJSONAs(t, router, server, 7, http.MethodPatch, "/me", body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body)
			}
			if repo.user.FullName() != "Alice" || repo.user.UserName() != "alice" {
				t.Error("user changed despite invalid input")
			}
		})
	}
}

func TestUpdateMeHandler_RepoFailureIs500(t *testing.T) {
	router, server, repo := accountTestRouter(t)
	repo.updateErr = errors.New("connection refused")

	w := sendJSONAs(t, router, server, 7, http.MethodPatch, "/me", `{"full_name":"X"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

// --- PUT /users/me/password

func TestChangePasswordHandler_ThenLoginWithNewPassword(t *testing.T) {
	router, server, _ := accountTestRouter(t)

	w := sendJSONAs(t, router, server, 7, http.MethodPut, "/me/password",
		`{"current_password":"s3cret!","new_password":"n3w-pass"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", w.Code, w.Body)
	}

	if w := postLogin(router, `{"username":"alice","password":"n3w-pass"}`); w.Code != http.StatusOK {
		t.Errorf("login with new password: status = %d, want 200", w.Code)
	}
	if w := postLogin(router, `{"username":"alice","password":"s3cret!"}`); w.Code != http.StatusUnauthorized {
		t.Errorf("login with old password: status = %d, want 401", w.Code)
	}
}

func TestChangePasswordHandler_WrongCurrentPassword(t *testing.T) {
	router, server, repo := accountTestRouter(t)
	before := repo.user.PasswordHash()

	w := sendJSONAs(t, router, server, 7, http.MethodPut, "/me/password",
		`{"current_password":"wrong-pass","new_password":"n3w-pass"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"invalid credentials"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	if repo.user.PasswordHash() != before {
		t.Error("password changed despite wrong current password")
	}
}

func TestChangePasswordHandler_Invalid(t *testing.T) {
	cases := map[string]string{
		"missing current": `{"new_password":"n3w-pass"}`,
		"missing new":     `{"current_password":"s3cret!"}`,
		"new too short":   `{"current_password":"s3cret!","new_password":"12345"}`,
		"new too long":    `{"current_password":"s3cret!","new_password":"` + strings.Repeat("p", 129) + `"}`,
		"malformed json":  `{"current_password":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			router, server, repo := accountTestRouter(t)
			before := repo.user.PasswordHash()
			w := sendJSONAs(t, router, server, 7, http.MethodPut, "/me/password", body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body)
			}
			if repo.user.PasswordHash() != before {
				t.Error("password changed despite invalid input")
			}
		})
	}
}

func TestAccountRoutes_RequireAuth(t *testing.T) {
	router, _, _ := accountTestRouter(t)
	for _, r := range []struct{ method, path, body string }{
		{http.MethodPatch, "/me", `{"full_name":"X"}`},
		{http.MethodPut, "/me/password", `{"current_password":"s3cret!","new_password":"n3w-pass"}`},
	} {
		req := httptest.NewRequest(r.method, r.path, bytes.NewBufferString(r.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", r.method, r.path, w.Code)
		}
	}
}
