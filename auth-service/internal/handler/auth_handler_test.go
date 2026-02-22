package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eaglebank/shared/cqrs"
	"github.com/gin-gonic/gin"
)

// ---- mock implementation ----

type mockAuthQuerier struct {
	loginFn   func(cqrs.LoginCommand) (string, error)
	refreshFn func(cqrs.RefreshTokenCommand) (string, error)
}

func (m *mockAuthQuerier) Login(cmd cqrs.LoginCommand) (string, error) {
	if m.loginFn != nil {
		return m.loginFn(cmd)
	}
	return "", fmt.Errorf("not configured")
}
func (m *mockAuthQuerier) RefreshToken(cmd cqrs.RefreshTokenCommand) (string, error) {
	if m.refreshFn != nil {
		return m.refreshFn(cmd)
	}
	return "", fmt.Errorf("not configured")
}

// ---- helper ----

func newAuthTestRouter(qrys AuthQuerier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAuthHandler(qrys)
	v1 := r.Group("/v1/auth")
	v1.POST("/login", h.Login)
	v1.POST("/refresh", h.RefreshToken)
	return r
}

func authDoRequest(router *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, url, nil)
	if body != nil {
		b, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, strings.NewReader(string(b)))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ---- tests ----

func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		body            interface{}
		loginFn        func(cqrs.LoginCommand) (string, error)
		expectedStatus int
	}{
		{
			name: "success - valid credentials return JWT",
			body: map[string]string{"email": "alice@example.com", "password": "securepass123"},
			loginFn: func(cmd cqrs.LoginCommand) (string, error) { return "mock.jwt.token", nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorised - invalid credentials",
			body: map[string]string{"email": "alice@example.com", "password": "wrongpass"},
			loginFn: func(cmd cqrs.LoginCommand) (string, error) { return "", fmt.Errorf("invalid credentials") },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "bad request - missing password",
			body:           map[string]string{"email": "alice@example.com"},
			loginFn:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bad request - missing email",
			body:           map[string]string{"password": "securepass123"},
			loginFn:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bad request - invalid email format",
			body:           map[string]string{"email": "not-an-email", "password": "securepass123"},
			loginFn:        nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newAuthTestRouter(&mockAuthQuerier{loginFn: tt.loginFn})
			w := authDoRequest(router, http.MethodPost, "/v1/auth/login", tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		body            interface{}
		refreshFn      func(cqrs.RefreshTokenCommand) (string, error)
		expectedStatus int
	}{
		{
			name: "success - valid token returns new JWT",
			body: map[string]string{"token": "valid.jwt.token"},
			refreshFn: func(cmd cqrs.RefreshTokenCommand) (string, error) { return "new.jwt.token", nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorised - invalid token",
			body: map[string]string{"token": "invalid.jwt.token"},
			refreshFn: func(cmd cqrs.RefreshTokenCommand) (string, error) { return "", fmt.Errorf("invalid token") },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "bad request - missing token field",
			body:      map[string]string{},
			refreshFn: nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newAuthTestRouter(&mockAuthQuerier{refreshFn: tt.refreshFn})
			w := authDoRequest(router, http.MethodPost, "/v1/auth/refresh", tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
