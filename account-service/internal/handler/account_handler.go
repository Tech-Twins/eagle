package handler

import (
	"net/http"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/middleware"
	"github.com/eaglebank/shared/models"
	"github.com/gin-gonic/gin"
)

// AccountCommander defines the write-side operations used by AccountHandler.
type AccountCommander interface {
	CreateAccount(cqrs.CreateAccountCommand) (*models.Account, error)
	UpdateAccount(cqrs.UpdateAccountCommand) (*models.AccountView, error)
	DeleteAccount(cqrs.DeleteAccountCommand) error
}

// AccountQuerier defines the read-side operations used by AccountHandler.
type AccountQuerier interface {
	GetAccount(cqrs.GetAccountQuery) (*models.AccountView, error)
	ListAccounts(cqrs.ListAccountsQuery) ([]models.AccountView, error)
}

// AccountHandler handles account-related HTTP requests.
type AccountHandler struct {
	commands AccountCommander
	queries  AccountQuerier
}

type CreateAccountRequest struct {
	Name        string `json:"name" validate:"required"`
	AccountType string `json:"accountType" validate:"required,oneof=personal"`
}

type UpdateAccountRequest struct {
	Name        string `json:"name"`
	AccountType string `json:"accountType" validate:"omitempty,oneof=personal"`
}

type ListAccountsResponse struct {
	Accounts []any `json:"accounts"`
}

func NewAccountHandler(commands AccountCommander, queries AccountQuerier) *AccountHandler {
	return &AccountHandler{commands: commands, queries: queries}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	account, err := h.commands.CreateAccount(cqrs.CreateAccountCommand{
		UserID:      userID,
		Name:        req.Name,
		AccountType: req.AccountType,
	})
	if err != nil {
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to create account")
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *AccountHandler) ListAccounts(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	views, err := h.queries.ListAccounts(cqrs.ListAccountsQuery{UserID: userID})
	if err != nil {
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to list accounts")
		return
	}

	accountsAny := make([]any, len(views))
	for i, v := range views {
		accountsAny[i] = v
	}
	c.JSON(http.StatusOK, ListAccountsResponse{Accounts: accountsAny})
}

func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountNumber := c.Param("accountNumber")
	userID, _ := middleware.GetUserID(c)

	view, err := h.queries.GetAccount(cqrs.GetAccountQuery{
		AccountNumber:    accountNumber,
		RequestingUserID: userID,
	})
	if err != nil {
		if err.Error() == "forbidden" {
			middleware.RespondWithError(c, http.StatusForbidden, "You can only access your own accounts")
			return
		}
		middleware.RespondWithError(c, http.StatusNotFound, "Account not found")
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	accountNumber := c.Param("accountNumber")
	userID, _ := middleware.GetUserID(c)

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	view, err := h.commands.UpdateAccount(cqrs.UpdateAccountCommand{
		AccountNumber:    accountNumber,
		RequestingUserID: userID,
		Name:             req.Name,
		AccountType:      req.AccountType,
	})
	if err != nil {
		if err.Error() == "account not found" {
			middleware.RespondWithError(c, http.StatusNotFound, "Account not found")
			return
		}
		if err.Error() == "forbidden" {
			middleware.RespondWithError(c, http.StatusForbidden, "You can only update your own accounts")
			return
		}
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to update account")
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	accountNumber := c.Param("accountNumber")
	userID, _ := middleware.GetUserID(c)

	err := h.commands.DeleteAccount(cqrs.DeleteAccountCommand{
		AccountNumber:    accountNumber,
		RequestingUserID: userID,
	})
	if err != nil {
		if err.Error() == "account not found" {
			middleware.RespondWithError(c, http.StatusNotFound, "Account not found")
			return
		}
		if err.Error() == "forbidden" {
			middleware.RespondWithError(c, http.StatusForbidden, "You can only delete your own accounts")
			return
		}
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to delete account")
		return
	}

	c.Status(http.StatusNoContent)
}
