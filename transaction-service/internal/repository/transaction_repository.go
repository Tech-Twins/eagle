package repository

import (
	"database/sql"
	"fmt"

	"github.com/eaglebank/shared/models"
)

// TransactionWriteRepository handles all state-mutating operations for transactions.
// It operates exclusively against the PostgreSQL write store (source of truth).
type TransactionWriteRepository struct {
	db *sql.DB
}

func NewTransactionWriteRepository(db *sql.DB) *TransactionWriteRepository {
	return &TransactionWriteRepository{db: db}
}

func (r *TransactionWriteRepository) Create(transaction *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, account_number, user_id, amount, currency, type, reference, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(query,
		transaction.ID, transaction.AccountNumber, transaction.UserID,
		transaction.Amount, transaction.Currency, transaction.Type,
		nullString(transaction.Reference), transaction.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
