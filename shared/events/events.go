package events

import "time"

// Event types
const (
	UserCreated = "user.created"
	UserUpdated = "user.updated"
	UserDeleted = "user.deleted"

	AccountCreated = "account.created"
	AccountUpdated = "account.updated"
	AccountDeleted = "account.deleted"

	TransactionCreated = "transaction.created"
	BalanceUpdated     = "balance.updated"
)

// Stream names
const (
	UserEventsStream        = "user.events"
	AccountEventsStream     = "account.events"
	TransactionEventsStream = "transaction.events"
)

// Base event structure
type Event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// User events
type UserCreatedEvent struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type UserUpdatedEvent struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type UserDeletedEvent struct {
	UserID string `json:"userId"`
}

// Account events
type AccountCreatedEvent struct {
	AccountNumber string `json:"accountNumber"`
	UserID        string `json:"userId"`
	Name          string `json:"name"`
	AccountType   string `json:"accountType"`
}

type AccountUpdatedEvent struct {
	AccountNumber string `json:"accountNumber"`
	UserID        string `json:"userId"`
	Name          string `json:"name"`
}

type AccountDeletedEvent struct {
	AccountNumber string `json:"accountNumber"`
	UserID        string `json:"userId"`
}

// Transaction events
type TransactionCreatedEvent struct {
	TransactionID string  `json:"transactionId"`
	AccountNumber string  `json:"accountNumber"`
	UserID        string  `json:"userId"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"`
	Currency      string  `json:"currency"`
}

type BalanceUpdatedEvent struct {
	AccountNumber string  `json:"accountNumber"`
	NewBalance    float64 `json:"newBalance"`
	Change        float64 `json:"change"`
}
