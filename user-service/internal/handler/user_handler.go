package handler

import (
	"net/http"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/middleware"
	"github.com/eaglebank/shared/models"
	"github.com/gin-gonic/gin"
)

// UserCommander defines the write-side operations used by UserHandler.
type UserCommander interface {
	CreateUser(cqrs.CreateUserCommand) (*models.User, error)
	UpdateUser(cqrs.UpdateUserCommand) (*models.UserView, error)
	DeleteUser(cqrs.DeleteUserCommand) error
}

// UserQuerier defines the read-side operations used by UserHandler.
type UserQuerier interface {
	GetUser(cqrs.GetUserQuery) (*models.UserView, error)
}

// UserHandler routes requests to the command or query service as appropriate.
type UserHandler struct {
	commands UserCommander
	queries  UserQuerier
}

type CreateUserRequest struct {
	Name        string         `json:"name" validate:"required"`
	Email       string         `json:"email" validate:"required,email"`
	Password    string         `json:"password" validate:"required,min=8"`
	PhoneNumber string         `json:"phoneNumber" validate:"required"`
	Address     models.Address `json:"address" validate:"required"`
}

type UpdateUserRequest struct {
	Name        string         `json:"name" validate:"required"`
	Email       string         `json:"email" validate:"required,email"`
	PhoneNumber string         `json:"phoneNumber" validate:"required"`
	Address     models.Address `json:"address" validate:"required"`
}

func NewUserHandler(commands UserCommander, queries UserQuerier) *UserHandler {
	return &UserHandler{commands: commands, queries: queries}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	user, err := h.commands.CreateUser(cqrs.CreateUserCommand{
		Name:        req.Name,
		Email:       req.Email,
		Password:    req.Password,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
	})
	if err != nil {
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("userId")
	requestingUserID, _ := middleware.GetUserID(c)

	view, err := h.queries.GetUser(cqrs.GetUserQuery{
		UserID:           userID,
		RequestingUserID: requestingUserID,
	})
	if err != nil {
		if err.Error() == "forbidden" {
			middleware.RespondWithError(c, http.StatusForbidden, "You can only access your own user details")
			return
		}
		middleware.RespondWithError(c, http.StatusNotFound, "User not found")
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("userId")
	requestingUserID, _ := middleware.GetUserID(c)

	if userID != requestingUserID {
		middleware.RespondWithError(c, http.StatusForbidden, "You can only update your own user details")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	view, err := h.commands.UpdateUser(cqrs.UpdateUserCommand{
		UserID:      userID,
		Name:        req.Name,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
	})
	if err != nil {
		if err.Error() == "user not found" {
			middleware.RespondWithError(c, http.StatusNotFound, "User not found")
			return
		}
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("userId")
	requestingUserID, _ := middleware.GetUserID(c)

	if userID != requestingUserID {
		middleware.RespondWithError(c, http.StatusForbidden, "You can only delete your own account")
		return
	}

	err := h.commands.DeleteUser(cqrs.DeleteUserCommand{UserID: userID})
	if err != nil {
		if err.Error() == "user not found" {
			middleware.RespondWithError(c, http.StatusNotFound, "User not found")
			return
		}
		if err.Error() == "user has active accounts" {
			middleware.RespondWithError(c, http.StatusConflict, "Cannot delete user with active bank accounts")
			return
		}
		middleware.RespondWithError(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	c.Status(http.StatusNoContent)
}
