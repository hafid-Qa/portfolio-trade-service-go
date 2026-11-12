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

	"app/internal/domain"
	"app/internal/password"

	"github.com/gin-gonic/gin"
)

// registerUserRepo records what Create was given; createErr simulates the
// DB rejecting the insert.
type registerUserRepo struct {
	loginUserRepo
	createdHash string
	createErr   error
}

func (r *registerUserRepo) Create(ctx context.Context, fullName, userName, email, passwordHash string) (domain.UserResult, error) {
	if r.createErr != nil {
		return domain.UserResult{}, r.createErr
	}
	r.createdHash = passwordHash
	now := time.Now()
	return domain.NewUserResult(1, fullName, strings.ToLower(userName), strings.ToLower(email), passwordHash, nil, now, now), nil
}

func registerTestRouter(repo *registerUserRepo) *gin.Engine {
	server := &Server{userService: domain.NewUserService(repo, password.Argon2Hasher{})}
	router := gin.New()
	router.POST("/users", server.CreateUserHandler)
	return router
}

func postUser(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

const validUserBody = `{"full_name":"Alice Smith","username":"alice","email":"alice@example.com","password":"s3cret!"}`

func TestCreateUserHandler_Created(t *testing.T) {
	repo := &registerUserRepo{}
	w := postUser(registerTestRouter(repo), validUserBody)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body)
	}
	var resp UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if resp.ID != 1 || resp.Username != "alice" || resp.Email != "alice@example.com" || resp.FullName != "Alice Smith" {
		t.Errorf("response = %+v", resp)
	}

	// The stored password is a real argon2id hash of what was sent...
	if err := password.Verify(repo.createdHash, "s3cret!"); err != nil {
		t.Errorf("stored hash doesn't verify: %v (hash %q)", err, repo.createdHash)
	}
	// ...and neither it nor the plaintext ever appears in the response.
	for _, secret := range []string{"s3cret!", repo.createdHash, "password"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("response body contains %q: %s", secret, w.Body)
		}
	}
}

func TestCreateUserHandler_Duplicate(t *testing.T) {
	repo := &registerUserRepo{createErr: fmt.Errorf("%w: users_username_key", domain.ErrUserAlreadyExists)}
	w := postUser(registerTestRouter(repo), validUserBody)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	// Generic: doesn't reveal which field (or the constraint name).
	if got, want := w.Body.String(), `{"error":"username or email already taken"}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestCreateUserHandler_RepoFailureIs500(t *testing.T) {
	repo := &registerUserRepo{createErr: errors.New("connection refused")}
	w := postUser(registerTestRouter(repo), validUserBody)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if got, want := w.Body.String(), `{"error":"internal server error"}`; got != want {
		t.Errorf("body = %s, want %s (no internal detail)", got, want)
	}
}

func TestCreateUserHandler_Validation(t *testing.T) {
	cases := map[string]string{
		"missing full_name":  `{"username":"alice","email":"alice@example.com","password":"s3cret!"}`,
		"non-alnum username": `{"full_name":"A","username":"al ice","email":"alice@example.com","password":"s3cret!"}`,
		"username too long":  `{"full_name":"A","username":"` + strings.Repeat("a", 51) + `","email":"alice@example.com","password":"s3cret!"}`,
		"bad email":          `{"full_name":"A","username":"alice","email":"not-an-email","password":"s3cret!"}`,
		"password too short": `{"full_name":"A","username":"alice","email":"alice@example.com","password":"12345"}`,
		"password too long":  `{"full_name":"A","username":"alice","email":"alice@example.com","password":"` + strings.Repeat("p", 129) + `"}`,
		"full_name too long": `{"full_name":"` + strings.Repeat("a", 101) + `","username":"alice","email":"alice@example.com","password":"s3cret!"}`,
		"malformed json":     `{"full_name":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &registerUserRepo{}
			w := postUser(registerTestRouter(repo), body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body)
			}
			if repo.createdHash != "" {
				t.Error("user was created despite invalid input")
			}
		})
	}
}
