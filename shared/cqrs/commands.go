package cqrs

import "github.com/eaglebank/shared/models"

type CreateUserCommand struct {
	Name        string
	Email       string
	Password    string
	PhoneNumber string
	Address     models.Address
}

type UpdateUserCommand struct {
	UserID      string
	Name        string
	Email       string
	PhoneNumber string
	Address     models.Address
}

type DeleteUserCommand struct {
	UserID string
}

type CreateAccountCommand struct {
	UserID      string
	Name        string
	AccountType string
}

type UpdateAccountCommand struct {
	AccountNumber    string
	RequestingUserID string
	Name             string
	AccountType      string
}

type DeleteAccountCommand struct {
	AccountNumber    string
	RequestingUserID string
}

type CreateTransactionCommand struct {
	AccountNumber string
	UserID        string
	Amount        float64
	Currency      string
	Type          string
	Reference     string
}

type LoginCommand struct {
	Email    string
	Password string
}

type RefreshTokenCommand struct {
	Token string
}
