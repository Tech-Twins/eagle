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

type mockTransactionCommander struct {
	createFn func(cqrs.CreateTransactionCommand) (*models.Transaction, error)
}

func (m *mockTransactionCommander) CreateTransaction(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) {
	if m.createFn != nil {
		return m.createFn(cmd)
	}
	return nil, fmt.Errorf("not configured")
}

type mockTransactionQuerier struct {
	getFn  func(cqrs.GetTransactionQuery) (*models.TransactionView, error)
	listFn func(cqrs.ListTransactionsQuery) ([]models.TransactionView, error)
}

func (m *mockTransactionQuerier) GetTransaction(q cqrs.GetTransactionQuery) (*models.TransactionView, error) {
	if m.getFn != nil {
		return m.getFn(q)
	}
	return nil, fmt.Errorf("not configured")
}
func (m *mockTransactionQuerier) ListTransactions(q cqrs.ListTransactionsQuery) ([]models.TransactionView, error) {
	if m.listFn != nil {
		return m.listFn(q)
	}
	return nil, fmt.Errorf("not configured")
}

// ---- helpers ----

func fakeAuthTx(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	}
}

func newTxTestRouter(cmds TransactionCommander, qrys TransactionQuerier, authUserID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(fakeAuthTx(authUserID))
	h := NewTransactionHandler(cmds, qrys)
	v1 := r.Group("/v1/accounts/:accountNumber/transactions")
	v1.POST("", h.CreateTransaction)
	v1.GET("", h.ListTransactions)
	v1.GET("/:transactionId", h.GetTransaction)
	return r
}

func txDoRequest(router *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
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

var txTestTransaction = &models.Transaction{
	ID: "tan-001", AccountNumber: "12345678", UserID: "usr-001",
	Amount: 50.00, Currency: "GBP", Type: "deposit",
	CreatedAt: time.Now(),
}

var txTestView = &models.TransactionView{
	ID: "tan-001", AccountNumber: "12345678", UserID: "usr-001",
	Amount: 50.00, Currency: "GBP", Type: "deposit",
	CreatedAt: time.Now(),
}

func txDepositBody() map[string]interface{} {
	return map[string]interface{}{"amount": 50.0, "currency": "GBP", "type": "deposit", "reference": "Test deposit"}
}

func txWithdrawalBody() map[string]interface{} {
	return map[string]interface{}{"amount": 25.0, "currency": "GBP", "type": "withdrawal", "reference": "Test withdrawal"}
}

// ---- tests ----

func TestCreateTransaction(t *testing.T) {
	tests := []struct {
		name           string
		accountNum     string
		body            interface{}
		createFn       func(cqrs.CreateTransactionCommand) (*models.Transaction, error)
		expectedStatus int
	}{
		{
			name: "success - deposit money into own account",
			accountNum: "12345678",
			body:           txDepositBody(),
			createFn:       func(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) { return txTestTransaction, nil },
			expectedStatus: http.StatusCreated,
		},
		{
			name: "success - withdraw money from own account",
			accountNum: "12345678",
			body:           txWithdrawalBody(),
			createFn:       func(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) { return txTestTransaction, nil },
			expectedStatus: http.StatusCreated,
		},
		{
			name: "unprocessable entity - insufficient funds",
			accountNum: "12345678",
			body:           txWithdrawalBody(),
			createFn:       func(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) { return nil, fmt.Errorf("insufficient funds") },
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "forbidden - transact on another user's account",
			accountNum: "99999999",
			body:           txDepositBody(),
			createFn:       func(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) { return nil, fmt.Errorf("forbidden") },
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - account does not exist",
			accountNum: "00000000",
			body:           txDepositBody(),
			createFn:       func(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) { return nil, fmt.Errorf("account not found") },
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "bad request - missing required fields",
			accountNum: "12345678",
			body:           map[string]interface{}{},
			createFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - amount is zero",
			accountNum: "12345678",
			body:           map[string]interface{}{"amount": 0, "currency": "GBP", "type": "deposit"},
			createFn:       nil,
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := &mockTransactionCommander{createFn: tt.createFn}
			router := newTxTestRouter(cmds, &mockTransactionQuerier{}, "usr-001")
			url := "/v1/accounts/" + tt.accountNum + "/transactions"
			w := txDoRequest(router, http.MethodPost, url, tt.body)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestListTransactions(t *testing.T) {
	tests := []struct {
		name           string
		accountNum     string
		listFn         func(cqrs.ListTransactionsQuery) ([]models.TransactionView, error)
		expectedStatus int
	}{
		{
			name: "success - list transactions on own account",
			accountNum: "12345678",
			listFn: func(q cqrs.ListTransactionsQuery) ([]models.TransactionView, error) {
				return []models.TransactionView{*txTestView}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden - list transactions on another user's account",
			accountNum: "99999999",
			listFn: func(q cqrs.ListTransactionsQuery) ([]models.TransactionView, error) {
				return nil, fmt.Errorf("forbidden")
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - account does not exist",
			accountNum: "00000000",
			listFn: func(q cqrs.ListTransactionsQuery) ([]models.TransactionView, error) {
				return nil, fmt.Errorf("account not found")
			},
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTxTestRouter(&mockTransactionCommander{}, &mockTransactionQuerier{listFn: tt.listFn}, "usr-001")
			url := "/v1/accounts/" + tt.accountNum + "/transactions"
			w := txDoRequest(router, http.MethodGet, url, nil)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestGetTransaction(t *testing.T) {
	tests := []struct {
		name           string
		accountNum     string
		transactionID  string
		getFn          func(cqrs.GetTransactionQuery) (*models.TransactionView, error)
		expectedStatus int
	}{
		{
			name: "success - fetch transaction on own account",
			accountNum: "12345678", transactionID: "tan-001",
			getFn: func(q cqrs.GetTransactionQuery) (*models.TransactionView, error) { return txTestView, nil },
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden - fetch transaction on another user's account",
			accountNum: "99999999", transactionID: "tan-001",
			getFn: func(q cqrs.GetTransactionQuery) (*models.TransactionView, error) { return nil, fmt.Errorf("forbidden") },
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "not found - transaction does not exist",
			accountNum: "12345678", transactionID: "tan-999",
			getFn: func(q cqrs.GetTransactionQuery) (*models.TransactionView, error) { return nil, fmt.Errorf("transaction not found") },
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "not found - account does not exist",
			accountNum: "00000000", transactionID: "tan-001",
			getFn: func(q cqrs.GetTransactionQuery) (*models.TransactionView, error) { return nil, fmt.Errorf("transaction not found") },
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "not found - transaction belongs to different account",
			accountNum: "12345678", transactionID: "tan-other",
			getFn: func(q cqrs.GetTransactionQuery) (*models.TransactionView, error) { return nil, fmt.Errorf("transaction not found") },
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTxTestRouter(&mockTransactionCommander{}, &mockTransactionQuerier{getFn: tt.getFn}, "usr-001")
			url := "/v1/accounts/" + tt.accountNum + "/transactions/" + tt.transactionID
			w := txDoRequest(router, http.MethodGet, url, nil)
			if w.Code != tt.expectedStatus {
				t.Errorf("[%s] expected %d got %d; body: %s", tt.name, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
