package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AccountRepository fetches account data from the account service via Redis cache or direct query
type AccountRepository struct {
	db    interface{} // Placeholder for potential direct DB access
	redis *redis.Client
}

type Account struct {
	AccountNumber string  `json:"accountNumber"`
	UserID        string  `json:"userId"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

func NewAccountRepository(db interface{}, redis *redis.Client) *AccountRepository {
	return &AccountRepository{
		db:    db,
		redis: redis,
	}
}

// GetAccount retrieves account information
// In a real system, this would make an HTTP call to the account service
// For simplicity, we'll cache account data in Redis during transaction creation
func (r *AccountRepository) GetAccount(ctx context.Context, accountNumber string) (*Account, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("account:%s", accountNumber)
	data, err := r.redis.Get(ctx, cacheKey).Result()

	if err == redis.Nil {
		// In production, make HTTP call to account service here
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account from cache: %w", err)
	}

	var account Account
	if err := json.Unmarshal([]byte(data), &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	return &account, nil
}

// CacheAccount stores account information in Redis cache
func (r *AccountRepository) CacheAccount(ctx context.Context, account *Account) error {
	cacheKey := fmt.Sprintf("account:%s", account.AccountNumber)
	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	// Cache for 5 minutes
	if err := r.redis.Set(ctx, cacheKey, data, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to cache account: %w", err)
	}

	return nil
}
