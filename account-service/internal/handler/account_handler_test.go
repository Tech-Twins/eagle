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

type mockAccountCommander struct {
	createFn func(cqrs.CreateAccountCommand) (*models.Account, error)
	updateFn func(cqrs.UpdateAccountCommand) (*models.AccountView, error)
	deleteFn func(cqrs.DeleteAccountCommand) error
}

func (m *mockAccountCommander) CreateAccount(cmd cqrs.CreateAccountCommand) (*models.Account, error) {
	if m.createFn != nil {
		return m.createFn(cmd)
	}
	return nil, fmt.Errorf("not configured")
}
func (m *mockAccountCommander) UpdateAccount(cmd cqrs.UpdateAccountCommand) (*models.AccountView, error) {
	if m.updateFn != nil {
		return m.updateFn(cmd)
	}
	return nil, fmt.Errorf("not configured")
}
func (m *mockAccountCommander) DeleteAccount(cmd cqrs.DeleteAccountCommand) error {
	if m.deleteFn != nil {
		return m.deleteFn(cmd)
	}
	return fmt.Errorf("not configured")
}

type mockAccountQuerier struct {
	getFn  func(cqrs.GetAccountQuery) (*models.AccountView, error)
	listFn func(cqrs.ListAccountsQuery) ([]models.AccountView, error)
}

func (m *mockAccountQuerier) GetAccount(q cqrs.GetAccountQuery) (*models.AccountView, error) {
	if m.getFn != nil {
		return m.getFn(q)
	}
	return nil, fmt.Errorf("not configured")
}
func (m *mockAccountQuerier) ListAccounts(q cqrs.ListAccountsQuery) ([]models.AccountView, error) {
	if m.listFn != nil {
		return m.listFn(q)
	}
	return nil, fmt.Errorf("not configured")
}

// ---- helpers ----

func fakeAuthAccount(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	}
}

func newAccountTestRouter(cmds AccountCommander, qrys AccountQuerier, authUserID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(fakeAuthAccount(authUserID))
	h := NewAccountHandler(cmds, qrys)
	v1 := r.Group("/v1/accounts")
	v1.POST("", h.CreateAccount)
	v1.GET("", h.ListAccounts)
	v1.GET("/:accountNumber", h.GetAccount)
	v1.PATCH("/:accountNumber", h.UpdateAccount)
	v1.DELETE("/:accountNumber", h.DeleteAccount)
	return r
}

func acctDoRequest(router *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
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

var aTestAccount = &models.Account{
	AccountNumber: "12345678", UserID: "usr-001", SortCode: "10-10-10",
	Name: "My Account", AccountType: "personal",
	Balance: 100.00, Currency: "GBP",
	CreatedAt: time.Now(), UpdatedAt: time.Now(),
}

var aTestAccountView = &models.AccountView{
	AccountNumber: "12345678", UserID: "usr-001", SortCode: "10-10-10",
	Name: "My Account", AccountType: "personal",
	Balance: 100.00, Currency: "GBP",
	CreatedAt: time.Now(), UpdatedAt: time.Now(),
}

func aValidCreateBody() map[string]interface{} {
	return map[string]interface{}{"name": "My Account", "accountType": "personal"}
}

func aValidUpdateBody() map[string]interface{} {
	return map[string]interface{}{"name": "My Updated Account", "accountType": "personal"}
}

// ---- tests ----

func TestCreateAccount(t *testing.T) {
	tests := []struct {
		name           string
		body            interface{}
		createFn       func(cqrs.CreateAccountCommand) (*models.Account, error)
		expectedStatus int
	}{
		{
			name: "success - create bank account",
			body:     aValidCreateBody(),
			createFn: func(cmd cqrs.CreateAccountCommand) (*models.Account, error) { return aTestAccount, nil },
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "bad request - missing required fields",
			body:           map[string]interface{}{},
			createFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bad request - invalid account type",
			body:           map[string]interface{}{"name": "Test", "accountType": "business"},
			createFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockAccountCommander{createFn: tt.createFn}
			router := newAccountTestRouter(cmds, &mockAccountQuerier{}, "usr-001")
			w := acctDoRequest(router, http.MethodPost, "/v1/accounts", tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestListAccounts(t *testing.T) {
	views := []models.AccountView{*aTestAccountView}
	listFn := func(q cqrs.ListAccountsQuery) ([]models.AccountView, error) { return views, nil }
	router := newAccountTestRouter(&mockAccountCommander{}, &mockAccountQuerier{listFn: listFn}, "usr-001")
	w := acctDoRequest(router, http.MethodGet, "/v1/accounts", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestGetAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountNum     string
		getFn          func(cqrs.GetAccountQuery) (*models.AccountView, error)
		expectedStatus int
	}{
		{
			name: "success - fetch own bank account",
			accountNum: "12345678",
			getFn:          func(q cqrs.GetAccountQuery) (*models.AccountView, error) { return aTestAccountView, nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden - fetch another user's bank account",
			accountNum: "99999999",
			getFn:          func(q cqrs.GetAccountQuery) (*models.AccountView, error) { return nil, fmt.Errorf("forbidden") },
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - account does not exist",
			accountNum: "00000000",
			getFn:          func(q cqrs.GetAccountQuery) (*models.AccountView, error) { return nil, fmt.Errorf("account not found") },
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newAccountTestRouter(&mockAccountCommander{}, &mockAccountQuerier{getFn: tt.getFn}, "usr-001")
			w := acctDoRequest(router, http.MethodGet, "/v1/accounts/"+tt.accountNum, nil)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestUpdateAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountNum     string
		body            interface{}
		updateFn       func(cqrs.UpdateAccountCommand) (*models.AccountView, error)
		expectedStatus int
	}{
		{
			name: "success - update own bank account",
			accountNum: "12345678",
			body:     aValidUpdateBody(),
			updateFn: func(cmd cqrs.UpdateAccountCommand) (*models.AccountView, error) { return aTestAccountView, nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden - update another user's bank account",
			accountNum: "99999999",
			body:     aValidUpdateBody(),
			updateFn: func(cmd cqrs.UpdateAccountCommand) (*models.AccountView, error) { return nil, fmt.Errorf("forbidden") },
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - account does not exist",
			accountNum: "00000000",
			body:     aValidUpdateBody(),
			updateFn: func(cmd cqrs.UpdateAccountCommand) (*models.AccountView, error) { return nil, fmt.Errorf("account not found") },
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockAccountCommander{updateFn: tt.updateFn}
			router := newAccountTestRouter(cmds, &mockAccountQuerier{}, "usr-001")
			w := acctDoRequest(router, http.MethodPatch, "/v1/accounts/"+tt.accountNum, tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountNum     string
		deleteFn       func(cqrs.DeleteAccountCommand) error
		expectedStatus int
	}{
		{
			name: "success - delete own bank account",
			accountNum: "12345678",
			deleteFn:       func(cmd cqrs.DeleteAccountCommand) error { return nil },
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "forbidden - delete another user's bank account",
			accountNum: "99999999",
			deleteFn:       func(cmd cqrs.DeleteAccountCommand) error { return fmt.Errorf("forbidden") },
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - account does not exist",
			accountNum: "00000000",
			deleteFn:       func(cmd cqrs.DeleteAccountCommand) error { return fmt.Errorf("account not found") },
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockAccountCommander{deleteFn: tt.deleteFn}
			router := newAccountTestRouter(cmds, &mockAccountQuerier{}, "usr-001")
			w := acctDoRequest(router, http.MethodDelete, "/v1/accounts/"+tt.accountNum, nil)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
