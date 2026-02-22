package repository

import (
	"database/sql"
	"fmt"

	"github.com/eaglebank/shared/models"
)

// AccountWriteRepository handles all state-mutating operations for accounts.
// It operates exclusively against the PostgreSQL write store (source of truth).
type AccountWriteRepository struct {
	db *sql.DB
}

func NewAccountWriteRepository(db *sql.DB) *AccountWriteRepository {
	return &AccountWriteRepository{db: db}
}

func (r *AccountWriteRepository) Create(account *models.Account) error {
	query := `
		INSERT INTO accounts (account_number, user_id, sort_code, name, account_type, balance, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(query,
		account.AccountNumber, account.UserID, account.SortCode, account.Name,
		account.AccountType, account.Balance, account.Currency,
		account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

// GetByAccountNumber fetches the full write model including UserID for ownership checks.
func (r *AccountWriteRepository) GetByAccountNumber(accountNumber string) (*models.Account, error) {
	query := `
		SELECT account_number, user_id, sort_code, name, account_type, balance, currency, created_at, updated_at
		FROM accounts
		WHERE account_number = $1 AND deleted_at IS NULL
	`
	var account models.Account
	err := r.db.QueryRow(query, accountNumber).Scan(
		&account.AccountNumber, &account.UserID, &account.SortCode, &account.Name,
		&account.AccountType, &account.Balance, &account.Currency,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return &account, nil
}

func (r *AccountWriteRepository) Update(account *models.Account) error {
	query := `
		UPDATE accounts
		SET name = $2, account_type = $3, updated_at = $4
		WHERE account_number = $1 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(query, account.AccountNumber, account.Name, account.AccountType, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (r *AccountWriteRepository) UpdateBalance(accountNumber string, newBalance float64) error {
	query := `
		UPDATE accounts
		SET balance = $2, updated_at = NOW()
		WHERE account_number = $1 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(query, accountNumber, newBalance)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (r *AccountWriteRepository) Delete(accountNumber string) error {
	query := `UPDATE accounts SET deleted_at = NOW() WHERE account_number = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(query, accountNumber)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (r *AccountWriteRepository) CountByUserID(userID string) (int, error) {
	query := `SELECT COUNT(*) FROM accounts WHERE user_id = $1 AND deleted_at IS NULL`
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count accounts: %w", err)
	}
	return count, nil
}
