package handler

import (
	"net/http"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/middleware"
	"github.com/eaglebank/shared/models"
	"github.com/gin-gonic/gin"
)

// TransactionCommander defines the write-side operations used by TransactionHandler.
type TransactionCommander interface {
	CreateTransaction(cqrs.CreateTransactionCommand) (*models.Transaction, error)
}

// TransactionQuerier defines the read-side operations used by TransactionHandler.
type TransactionQuerier interface {
	GetTransaction(cqrs.GetTransactionQuery) (*models.TransactionView, error)
	ListTransactions(cqrs.ListTransactionsQuery) ([]models.TransactionView, error)
}

type TransactionHandler struct {
	commands TransactionCommander
	queries  TransactionQuerier
}

type CreateTransactionRequest struct {
	Amount    float64 `json:"amount" validate:"required,gt=0"`
	Currency  string  `json:"currency" validate:"required,oneof=GBP"`
	Type      string  `json:"type" validate:"required,oneof=deposit withdrawal"`
	Reference string  `json:"reference"`
}

type ListTransactionsResponse struct {
	Transactions []any `json:"transactions"`
}

func NewTransactionHandler(commands TransactionCommander, queries TransactionQuerier) *TransactionHandler {
	return &TransactionHandler{commands: commands, queries: queries}
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	accountNumber := c.Param("accountNumber")
	userID, _ := middleware.GetUserID(c)

	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	transaction, err := h.commands.CreateTransaction(cqrs.CreateTransactionCommand{
		AccountNumber: accountNumber,
		UserID:        userID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Type:          req.Type,
		Reference:     req.Reference,
	})
	if err != nil {
		switch err.Error() {
		case "account not found":
			middleware.RespondWithError(c, http.StatusNotFound, "Account not found")
		case "forbidden":
			middleware.RespondWithError(c, http.StatusForbidden, "You can only create transactions for your own accounts")
		case "insufficient funds":
			middleware.RespondWithError(c, http.StatusUnprocessableEntity, "Insufficient funds")
		default:
			middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to create transaction")
		}
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	accountNumber := c.Param("accountNumber")
	userID, _ := middleware.GetUserID(c)

	views, err := h.queries.ListTransactions(cqrs.ListTransactionsQuery{
		AccountNumber: accountNumber,
		UserID:        userID,
	})
	if err != nil {
		switch err.Error() {
		case "account not found":
			middleware.RespondWithError(c, http.StatusNotFound, "Account not found")
		case "forbidden":
			middleware.RespondWithError(c, http.StatusForbidden, "You can only view transactions for your own accounts")
		default:
			middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to list transactions")
		}
		return
	}

	transactionsAny := make([]any, len(views))
	for i, v := range views {
		transactionsAny[i] = v
	}
	c.JSON(http.StatusOK, ListTransactionsResponse{Transactions: transactionsAny})
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	accountNumber := c.Param("accountNumber")
	transactionID := c.Param("transactionId")
	userID, _ := middleware.GetUserID(c)

	view, err := h.queries.GetTransaction(cqrs.GetTransactionQuery{
		TransactionID: transactionID,
		AccountNumber: accountNumber,
		UserID:        userID,
	})
	if err != nil {
		switch err.Error() {
		case "transaction not found":
			middleware.RespondWithError(c, http.StatusNotFound, "Transaction not found")
		case "forbidden":
			middleware.RespondWithError(c, http.StatusForbidden, "You can only view your own transactions")
		default:
			middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to get transaction")
		}
		return
	}

	c.JSON(http.StatusOK, view)
}
