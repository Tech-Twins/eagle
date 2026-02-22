package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/eaglebank/shared/models"
	sharedredis "github.com/eaglebank/shared/redis"
	goredis "github.com/redis/go-redis/v9"
)

const accountViewKeyPrefix = "account:view:"

// AccountReadRepository handles all read operations for accounts.
// It treats Redis as the primary read store (the CQRS read model) and falls
// back to PostgreSQL transparently, warming the cache on every cold read.
type AccountReadRepository struct {
	db    *sql.DB
	redis *goredis.Client
	cache *sharedredis.ViewCache[models.AccountView]
}

func NewAccountReadRepository(db *sql.DB, redisClient *goredis.Client) *AccountReadRepository {
	return &AccountReadRepository{
		db:    db,
		redis: redisClient,
		cache: sharedredis.NewViewCache[models.AccountView](redisClient, 0),
	}
}

// GetByAccountNumber returns an AccountView, trying Redis first then PostgreSQL.
func (r *AccountReadRepository) GetByAccountNumber(ctx context.Context, accountNumber string) (*models.AccountView, error) {
	cacheKey := accountViewKeyPrefix + accountNumber

	if view, ok := r.cache.Get(ctx, cacheKey); ok {
		return view, nil
	}

	// Fallback: PostgreSQL — include user_id so the service can enforce ownership.
	query := `
		SELECT account_number, user_id, sort_code, name, account_type, balance, currency, created_at, updated_at
		FROM accounts
		WHERE account_number = $1 AND deleted_at IS NULL
	`
	var view models.AccountView
	pgErr := r.db.QueryRow(query, accountNumber).Scan(
		&view.AccountNumber, &view.UserID, &view.SortCode, &view.Name,
		&view.AccountType, &view.Balance, &view.Currency,
		&view.CreatedAt, &view.UpdatedAt,
	)
	if pgErr == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if pgErr != nil {
		return nil, fmt.Errorf("failed to get account: %w", pgErr)
	}

	// Warm the cache
	r.CacheAccountView(ctx, &view)
	return &view, nil
}

// ListByUserID returns all AccountViews for the given user from PostgreSQL.
func (r *AccountReadRepository) ListByUserID(ctx context.Context, userID string) ([]models.AccountView, error) {
	query := `
		SELECT account_number, user_id, sort_code, name, account_type, balance, currency, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var views []models.AccountView
	for rows.Next() {
		var view models.AccountView
		if err := rows.Scan(
			&view.AccountNumber, &view.UserID, &view.SortCode, &view.Name,
			&view.AccountType, &view.Balance, &view.Currency,
			&view.CreatedAt, &view.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		views = append(views, view)
	}
	return views, nil
}

// CacheAccountView stores or refreshes the Redis read model for an account.
// Called by the command service after every mutation to keep the read model current.
func (r *AccountReadRepository) CacheAccountView(ctx context.Context, view *models.AccountView) {
	r.cache.Set(ctx, accountViewKeyPrefix+view.AccountNumber, view)
}

// InvalidateAccountView removes the Redis read model entry for a deleted account.
func (r *AccountReadRepository) InvalidateAccountView(ctx context.Context, accountNumber string) {
	r.cache.Delete(ctx, accountViewKeyPrefix+accountNumber)
}

const processedTxnKeyPrefix = "processed:txn:"

// IsTransactionProcessed returns true if this transaction ID has already been
// applied to a balance. Guards against duplicate delivery under at-least-once
// Redis Streams semantics.
func (r *AccountReadRepository) IsTransactionProcessed(ctx context.Context, transactionID string) bool {
	val, err := r.redis.Exists(ctx, processedTxnKeyPrefix+transactionID).Result()
	return err == nil && val > 0
}

// MarkTransactionProcessed records that a transaction has been applied.
// The key expires after 72 hours — long enough to cover any realistic
// redelivery window from a consumer group.
func (r *AccountReadRepository) MarkTransactionProcessed(ctx context.Context, transactionID string) {
	key := processedTxnKeyPrefix + transactionID
	if err := r.redis.Set(ctx, key, "1", 72*time.Hour).Err(); err != nil {
		log.Printf("Failed to mark transaction %s as processed: %v", transactionID, err)
	}
}
