package redis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ViewCache is a generic JSON-backed Redis cache for read model projections.
// Bind it to a specific view type T; each instance holds a Redis client and an
// optional TTL (pass 0 for keys that should not expire).
type ViewCache[T any] struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewViewCache creates a ViewCache backed by the provided Redis client.
func NewViewCache[T any](client *goredis.Client, ttl time.Duration) *ViewCache[T] {
	return &ViewCache[T]{client: client, ttl: ttl}
}

// Get retrieves and unmarshals a value from Redis.
// Returns (nil, false) on any miss or deserialisation error.
func (c *ViewCache[T]) Get(ctx context.Context, key string) (*T, bool) {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}
	var v T
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return nil, false
	}
	return &v, true
}

// Set marshals value and stores it in Redis under key.
// Errors are logged rather than returned — a cache write miss is non-fatal.
func (c *ViewCache[T]) Set(ctx context.Context, key string, value *T) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("ViewCache: marshal error for key %s: %v", key, err)
		return
	}
	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		log.Printf("ViewCache: write error for key %s: %v", key, err)
	}
}

// Delete removes a key from Redis.
func (c *ViewCache[T]) Delete(ctx context.Context, key string) {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		log.Printf("ViewCache: delete error for key %s: %v", key, err)
	}
}
