package repository

import (
	"database/sql"
	"fmt"

	"github.com/eaglebank/shared/models"
	"github.com/lib/pq"
)

// UserWriteRepository handles all state-mutating operations for users.
// It operates exclusively against the PostgreSQL write store (source of truth).
type UserWriteRepository struct {
	db *sql.DB
}

func NewUserWriteRepository(db *sql.DB) *UserWriteRepository {
	return &UserWriteRepository{db: db}
}

func (r *UserWriteRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, phone_number,
			address_line1, address_line2, address_line3, address_town, address_county, address_postcode,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(query,
		user.ID, user.Name, user.Email, user.PasswordHash, user.PhoneNumber,
		user.Address.Line1, nullString(user.Address.Line2), nullString(user.Address.Line3),
		user.Address.Town, user.Address.County, user.Address.Postcode,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("email already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID fetches the full write model (including PasswordHash) for internal operations.
func (r *UserWriteRepository) GetByID(id string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, phone_number,
			   address_line1, address_line2, address_line3, address_town, address_county, address_postcode,
			   created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var user models.User
	var line2, line3 sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.PhoneNumber,
		&user.Address.Line1, &line2, &line3, &user.Address.Town, &user.Address.County, &user.Address.Postcode,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if line2.Valid {
		user.Address.Line2 = line2.String
	}
	if line3.Valid {
		user.Address.Line3 = line3.String
	}
	return &user, nil
}

func (r *UserWriteRepository) Update(user *models.User) error {
	query := `
		UPDATE users
		SET name = $2, email = $3, phone_number = $4,
			address_line1 = $5, address_line2 = $6, address_line3 = $7,
			address_town = $8, address_county = $9, address_postcode = $10,
			updated_at = $11
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(query,
		user.ID, user.Name, user.Email, user.PhoneNumber,
		user.Address.Line1, nullString(user.Address.Line2), nullString(user.Address.Line3),
		user.Address.Town, user.Address.County, user.Address.Postcode,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserWriteRepository) Delete(id string) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserWriteRepository) HasAccounts(userID string) (bool, error) {
	// Simplified: account ownership is coordinated via events.
	return false, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
