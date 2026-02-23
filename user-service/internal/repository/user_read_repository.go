package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/eaglebank/shared/models"
	sharedredis "github.com/eaglebank/shared/redis"
	goredis "github.com/redis/go-redis/v9"
)

const userViewKeyPrefix = "user:view:"
const userAccCountKeyPrefix = "user:acc-count:"

// UserReadRepository handles all read operations for users.
// It uses Redis as the primary read store, falling back to PostgreSQL on a miss.
type UserReadRepository struct {
	db    *sql.DB
	redis *goredis.Client
	cache *sharedredis.ViewCache[models.UserView]
}

func NewUserReadRepository(db *sql.DB, redisClient *goredis.Client) *UserReadRepository {
	return &UserReadRepository{
		db:    db,
		redis: redisClient,
		cache: sharedredis.NewViewCache[models.UserView](redisClient, 0),
	}
}

// GetByID returns a UserView from Redis first, then PostgreSQL.
func (r *UserReadRepository) GetByID(ctx context.Context, id string) (*models.UserView, error) {
	cacheKey := userViewKeyPrefix + id

	if view, ok := r.cache.Get(ctx, cacheKey); ok {
		return view, nil
	}

	// Fallback: PostgreSQL
	query := `
		SELECT id, name, email, phone_number,
			   address_line1, address_line2, address_line3, address_town, address_county, address_postcode,
			   created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var view models.UserView
	var line2, line3 sql.NullString

	pgErr := r.db.QueryRow(query, id).Scan(
		&view.ID, &view.Name, &view.Email, &view.PhoneNumber,
		&view.Address.Line1, &line2, &line3, &view.Address.Town, &view.Address.County, &view.Address.Postcode,
		&view.CreatedAt, &view.UpdatedAt,
	)
	if pgErr == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if pgErr != nil {
		return nil, fmt.Errorf("failed to get user: %w", pgErr)
	}

	if line2.Valid {
		view.Address.Line2 = line2.String
	}
	if line3.Valid {
		view.Address.Line3 = line3.String
	}

	// Warm the cache
	r.CacheUserView(ctx, &view)
	return &view, nil
}

// CacheUserView stores or refreshes the Redis read model for a user.
// Called by the command service after every mutation.
func (r *UserReadRepository) CacheUserView(ctx context.Context, view *models.UserView) {
	r.cache.Set(ctx, userViewKeyPrefix+view.ID, view)
}

// InvalidateUserView removes the Redis read model entry for a deleted user.
func (r *UserReadRepository) InvalidateUserView(ctx context.Context, userID string) {
	r.cache.Delete(ctx, userViewKeyPrefix+userID)
}

// HasActiveAccounts returns true if the user has one or more open accounts,
// as tracked by account.created / account.deleted events in Redis.
func (r *UserReadRepository) HasActiveAccounts(ctx context.Context, userID string) bool {
	val, err := r.redis.Get(ctx, userAccCountKeyPrefix+userID).Int64()
	return err == nil && val > 0
}

// IncrAccountCount increments the active-account counter for a user.
func (r *UserReadRepository) IncrAccountCount(ctx context.Context, userID string) {
	r.redis.Incr(ctx, userAccCountKeyPrefix+userID)
}

// DecrAccountCount decrements the active-account counter for a user (floor 0).
func (r *UserReadRepository) DecrAccountCount(ctx context.Context, userID string) {
	current, err := r.redis.Get(ctx, userAccCountKeyPrefix+userID).Int64()
	if err == nil && current > 0 {
		r.redis.Decr(ctx, userAccCountKeyPrefix+userID)
	}
}
