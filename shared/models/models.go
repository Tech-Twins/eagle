package models

import "time"

type Address struct {
	Line1    string `json:"line1" validate:"required"`
	Line2    string `json:"line2,omitempty"`
	Line3    string `json:"line3,omitempty"`
	Town     string `json:"town" validate:"required"`
	County   string `json:"county" validate:"required"`
	Postcode string `json:"postcode" validate:"required"`
}

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	PhoneNumber  string    `json:"phoneNumber"`
	Address      Address   `json:"address"`
	CreatedAt    time.Time `json:"createdTimestamp"`
	UpdatedAt    time.Time `json:"updatedTimestamp"`
}

type Account struct {
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

type Transaction struct {
	ID            string    `json:"id"`
	AccountNumber string    `json:"-"`
	UserID        string    `json:"userId"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Type          string    `json:"type"`
	Reference     string    `json:"reference,omitempty"`
	CreatedAt     time.Time `json:"createdTimestamp"`
}
