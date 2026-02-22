package handler

import (
	"net/http"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/middleware"
	"github.com/gin-gonic/gin"
)

// AuthQuerier defines the read-side operations used by AuthHandler.
type AuthQuerier interface {
	Login(cqrs.LoginCommand) (string, error)
	RefreshToken(cqrs.RefreshTokenCommand) (string, error)
}

// AuthHandler handles login and token refresh. No command service needed.
type AuthHandler struct {
	queries AuthQuerier
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

func NewAuthHandler(queries AuthQuerier) *AuthHandler {
	return &AuthHandler{queries: queries}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	token, err := h.queries.Login(cqrs.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		middleware.RespondWithError(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	c.JSON(http.StatusOK, AuthResponse{Token: token})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if validationErrors := middleware.ValidateRequest(req); validationErrors != nil {
		middleware.RespondWithValidationError(c, validationErrors)
		return
	}

	token, err := h.queries.RefreshToken(cqrs.RefreshTokenCommand{
		Token: req.Token,
	})
	if err != nil {
		middleware.RespondWithError(c, http.StatusUnauthorized, "Invalid token")
		return
	}

	c.JSON(http.StatusOK, AuthResponse{Token: token})
}
