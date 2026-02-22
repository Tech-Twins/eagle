package models

import "time"

// UserView is the read-optimised projection of a user.
// It never exposes PasswordHash and may be extended with derived/denormalised fields.
type UserView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phoneNumber"`
	Address     Address   `json:"address"`
	CreatedAt   time.Time `json:"createdTimestamp"`
	UpdatedAt   time.Time `json:"updatedTimestamp"`
}

// AccountView is the read-optimised projection of an account.
// UserID is populated for ownership checks but never serialised to the API response.
type AccountView struct {
	AccountNumber string    `json:"accountNumber"`
	UserID        string    `json:"-"`
	SortCode      string    `json:"sortCode"`
	Name          string    `json:"name"`
	AccountType   string    `json:"accountType"`
	Balance       float64   `json:"balance"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"createdTimestamp"`
	UpdatedAt     time.Time `json:"updatedTimestamp"`
}

// TransactionView is the read-optimised projection of a transaction.
// UserID is populated for ownership checks but never serialised to the API response.
type TransactionView struct {
	ID            string    `json:"id"`
	AccountNumber string    `json:"accountNumber"`
	UserID        string    `json:"-"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Type          string    `json:"type"`
	Reference     string    `json:"reference,omitempty"`
	CreatedAt     time.Time `json:"createdTimestamp"`
}
