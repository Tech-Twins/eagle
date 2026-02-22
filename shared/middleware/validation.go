package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type BadRequestErrorResponse struct {
	Message string            `json:"message"`
	Details []ValidationError `json:"details"`
}

func ValidateRequest(obj any) []ValidationError {
	var validationErrors []ValidationError

	err := validate.Struct(obj)
	if err == nil {
		return nil
	}

	for _, err := range err.(validator.ValidationErrors) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   err.Field(),
			Message: getErrorMsg(err),
			Type:    err.Tag(),
		})
	}

	return validationErrors
}

func getErrorMsg(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "gt":
		return "Value must be greater than " + err.Param()
	case "gte":
		return "Value must be greater than or equal to " + err.Param()
	default:
		return "Invalid value"
	}
}

func RespondWithValidationError(c *gin.Context, validationErrors []ValidationError) {
	c.JSON(http.StatusBadRequest, BadRequestErrorResponse{
		Message: "Invalid request data",
		Details: validationErrors,
	})
}

func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"message": message,
	})
}
