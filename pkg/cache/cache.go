package cache

import (
	"context"
	"time"
)

// Cache is the interface for cache.
type Cache interface {
	// Get gets the value from cache and unmarshal it to v.
	Get(ctx context.Context, key string, v any) error
	// Set sets the value to cache.
	Set(ctx context.Context, key string, v any, ttl time.Duration) error
	// Has reports whether value for the given key exists.
	Has(ctx context.Context, key string) bool
	// Del deletes the value for the given key.
	Del(ctx context.Context, key string) error
	// Take takes the result from cache first, if not found,
	// call query to get a value(such as query database) and set cache using c.expiry, then return the result.
	Take(ctx context.Context, v any, key string, ttl time.Duration, query func() (any, error)) error
	// IsNotFound detect the error weather not found from cache
	IsNotFound(err error) bool
}
