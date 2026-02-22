package repository

import (
	"database/sql"
	"fmt"

	"github.com/eaglebank/shared/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, phone_number,
			   address_line1, address_line2, address_line3, address_town, address_county, address_postcode,
			   created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user models.User
	var line2, line3 sql.NullString

	err := r.db.QueryRow(query, email).Scan(
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
