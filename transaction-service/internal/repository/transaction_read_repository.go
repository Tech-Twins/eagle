package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/eaglebank/shared/models"
	sharedredis "github.com/eaglebank/shared/redis"
	goredis "github.com/redis/go-redis/v9"
)

const transactionViewKeyPrefix = "transaction:view:"

// TransactionReadRepository handles all read operations for transactions.
// It uses Redis as the primary read store, falling back to PostgreSQL on a miss.
type TransactionReadRepository struct {
	db    *sql.DB
	cache *sharedredis.ViewCache[models.TransactionView]
}

func NewTransactionReadRepository(db *sql.DB, redisClient *goredis.Client) *TransactionReadRepository {
	return &TransactionReadRepository{
		db:    db,
		cache: sharedredis.NewViewCache[models.TransactionView](redisClient, 0),
	}
}

// GetByID returns a TransactionView by attempting Redis first, then PostgreSQL.
func (r *TransactionReadRepository) GetByID(ctx context.Context, id, accountNumber string) (*models.TransactionView, error) {
	cacheKey := fmt.Sprintf("%s%s:%s", transactionViewKeyPrefix, accountNumber, id)
	if view, ok := r.cache.Get(ctx, cacheKey); ok {
		return view, nil
	}

	// Fallback: PostgreSQL
	query := `
		SELECT id, account_number, user_id, amount, currency, type, reference, created_at
		FROM transactions
		WHERE id = $1 AND account_number = $2
	`
	var view models.TransactionView
	var reference sql.NullString

	pgErr := r.db.QueryRow(query, id, accountNumber).Scan(
		&view.ID, &view.AccountNumber, &view.UserID,
		&view.Amount, &view.Currency, &view.Type,
		&reference, &view.CreatedAt,
	)
	if pgErr == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if pgErr != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", pgErr)
	}
	if reference.Valid {
		view.Reference = reference.String
	}

	// Warm the cache
	r.CacheTransactionView(ctx, &view)
	return &view, nil
}

// ListByAccountNumber returns all TransactionViews for an account from PostgreSQL.
func (r *TransactionReadRepository) ListByAccountNumber(ctx context.Context, accountNumber string) ([]models.TransactionView, error) {
	query := `
		SELECT id, account_number, user_id, amount, currency, type, reference, created_at
		FROM transactions
		WHERE account_number = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, accountNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var views []models.TransactionView
	for rows.Next() {
		var view models.TransactionView
		var reference sql.NullString

		if err := rows.Scan(
			&view.ID, &view.AccountNumber, &view.UserID,
			&view.Amount, &view.Currency, &view.Type,
			&reference, &view.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		if reference.Valid {
			view.Reference = reference.String
		}
		views = append(views, view)
	}
	return views, nil
}

// CacheTransactionView stores the read model for a transaction in Redis.
// Called by the command service immediately after a successful Create.
func (r *TransactionReadRepository) CacheTransactionView(ctx context.Context, view *models.TransactionView) {
	cacheKey := fmt.Sprintf("%s%s:%s", transactionViewKeyPrefix, view.AccountNumber, view.ID)
	r.cache.Set(ctx, cacheKey, view)
}
