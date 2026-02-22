package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/models"
	"github.com/gin-gonic/gin"
)

// ---- mock implementations ----

type mockUserCommander struct {
	createFn func(cqrs.CreateUserCommand) (*models.User, error)
	updateFn func(cqrs.UpdateUserCommand) (*models.UserView, error)
	deleteFn func(cqrs.DeleteUserCommand) error
}

func (m *mockUserCommander) CreateUser(cmd cqrs.CreateUserCommand) (*models.User, error) {
	if m.createFn != nil {
		return m.createFn(cmd)
	}
	return nil, fmt.Errorf("not configured")
}
func (m *mockUserCommander) UpdateUser(cmd cqrs.UpdateUserCommand) (*models.UserView, error) {
	if m.updateFn != nil {
		return m.updateFn(cmd)
	}
	return nil, fmt.Errorf("not configured")
}
func (m *mockUserCommander) DeleteUser(cmd cqrs.DeleteUserCommand) error {
	if m.deleteFn != nil {
		return m.deleteFn(cmd)
	}
	return fmt.Errorf("not configured")
}

type mockUserQuerier struct {
	getFn func(cqrs.GetUserQuery) (*models.UserView, error)
}

func (m *mockUserQuerier) GetUser(q cqrs.GetUserQuery) (*models.UserView, error) {
	if m.getFn != nil {
		return m.getFn(q)
	}
	return nil, fmt.Errorf("not configured")
}

// ---- helpers ----

func fakeAuthUser(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	}
}

func newUserTestRouter(cmds UserCommander, qrys UserQuerier, authUserID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(fakeAuthUser(authUserID))
	h := NewUserHandler(cmds, qrys)
	v1 := r.Group("/v1/users")
	v1.POST("", h.CreateUser)
	v1.GET("/:userId", h.GetUser)
	v1.PATCH("/:userId", h.UpdateUser)
	v1.DELETE("/:userId", h.DeleteUser)
	return r
}

func userDoRequest(router *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
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

// ---- test data ----

var uTestAddress = models.Address{
	Line1:    "1 Eagle Street",
	Town:     "London",
	County:   "Greater London",
	Postcode: "EC1A 1BB",
}

var uTestUserView = &models.UserView{
	ID: "usr-001", Name: "Alice", Email: "alice@example.com",
	PhoneNumber: "+441234567890", Address: uTestAddress,
	CreatedAt: time.Now(), UpdatedAt: time.Now(),
}

var uTestUser = &models.User{
	ID: "usr-001", Name: "Alice", Email: "alice@example.com",
	PhoneNumber: "+441234567890", Address: uTestAddress,
	CreatedAt: time.Now(), UpdatedAt: time.Now(),
}

func uValidCreateBody() map[string]interface{} {
	return map[string]interface{}{
		"name": "Alice Smith", "email": "alice@example.com",
		"password": "securepass123", "phoneNumber": "+441234567890",
		"address": map[string]string{
			"line1": "1 Eagle Street", "town": "London",
			"county": "Greater London", "postcode": "EC1A 1BB",
		},
	}
}

func uValidUpdateBody() map[string]interface{} {
	return map[string]interface{}{
		"name": "Alice Updated", "email": "alice@example.com",
		"phoneNumber": "+441234567890",
		"address": map[string]string{
			"line1": "1 Eagle Street", "town": "London",
			"county": "Greater London", "postcode": "EC1A 1BB",
		},
	}
}

// ---- tests ----

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		body            interface{}
		createFn       func(cqrs.CreateUserCommand) (*models.User, error)
		expectedStatus int
	}{
		{
			name:           "success - creates new user",
			body:           uValidCreateBody(),
			createFn:       func(cmd cqrs.CreateUserCommand) (*models.User, error) { return uTestUser, nil },
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "bad request - missing required fields",
			body:           map[string]interface{}{"email": "alice@example.com"},
			createFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bad request - invalid email format",
			body:           map[string]interface{}{"name": "Alice", "email": "not-valid", "password": "pass12345", "phoneNumber": "123"},
			createFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockUserCommander{createFn: tt.createFn}
			router := newUserTestRouter(cmds, &mockUserQuerier{}, "")
			w := userDoRequest(router, http.MethodPost, "/v1/users", tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name           string
		urlUserID      string
		authUserID     string
		getFn          func(cqrs.GetUserQuery) (*models.UserView, error)
		expectedStatus int
	}{
		{
			name: "success - fetch own user details",
			urlUserID: "usr-001", authUserID: "usr-001",
			getFn:          func(q cqrs.GetUserQuery) (*models.UserView, error) { return uTestUserView, nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden - fetch another user's details",
			urlUserID: "usr-002", authUserID: "usr-001",
			getFn:          func(q cqrs.GetUserQuery) (*models.UserView, error) { return nil, fmt.Errorf("forbidden") },
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - user does not exist",
			urlUserID: "usr-999", authUserID: "usr-999",
			getFn:          func(q cqrs.GetUserQuery) (*models.UserView, error) { return nil, fmt.Errorf("user not found") },
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newUserTestRouter(&mockUserCommander{}, &mockUserQuerier{getFn: tt.getFn}, tt.authUserID)
			w := userDoRequest(router, http.MethodGet, "/v1/users/"+tt.urlUserID, nil)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		urlUserID      string
		authUserID     string
		body            interface{}
		updateFn       func(cqrs.UpdateUserCommand) (*models.UserView, error)
		expectedStatus int
	}{
		{
			name: "success - update own user details",
			urlUserID: "usr-001", authUserID: "usr-001",
			body:           uValidUpdateBody(),
			updateFn:       func(cmd cqrs.UpdateUserCommand) (*models.UserView, error) { return uTestUserView, nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden - update another user's details",
			urlUserID: "usr-002", authUserID: "usr-001",
			body:           uValidUpdateBody(),
			updateFn:       nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - user does not exist",
			urlUserID: "usr-999", authUserID: "usr-999",
			body:           uValidUpdateBody(),
			updateFn:       func(cmd cqrs.UpdateUserCommand) (*models.UserView, error) { return nil, fmt.Errorf("user not found") },
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "bad request - missing required fields",
			urlUserID: "usr-001", authUserID: "usr-001",
			body:           map[string]interface{}{},
			updateFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockUserCommander{updateFn: tt.updateFn}
			router := newUserTestRouter(cmds, &mockUserQuerier{}, tt.authUserID)
			w := userDoRequest(router, http.MethodPatch, "/v1/users/"+tt.urlUserID, tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name           string
		urlUserID      string
		authUserID     string
		deleteFn       func(cqrs.DeleteUserCommand) error
		expectedStatus int
	}{
		{
			name: "success - delete own user with no bank accounts",
			urlUserID: "usr-001", authUserID: "usr-001",
			deleteFn:       func(cmd cqrs.DeleteUserCommand) error { return nil },
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "conflict - user has active bank accounts",
			urlUserID: "usr-001", authUserID: "usr-001",
			deleteFn:       func(cmd cqrs.DeleteUserCommand) error { return fmt.Errorf("user has active accounts") },
			expectedStatus: http.StatusConflict,
		},
		{
			name: "forbidden - delete another user",
			urlUserID: "usr-002", authUserID: "usr-001",
			deleteFn:       nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - user does not exist",
			urlUserID: "usr-999", authUserID: "usr-999",
			deleteFn:       func(cmd cqrs.DeleteUserCommand) error { return fmt.Errorf("user not found") },
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockUserCommander{deleteFn: tt.deleteFn}
			router := newUserTestRouter(cmds, &mockUserQuerier{}, tt.authUserID)
			w := userDoRequest(router, http.MethodDelete, "/v1/users/"+tt.urlUserID, nil)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
