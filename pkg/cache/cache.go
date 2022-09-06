package cache

import (
	"time"
)

type Cache interface {
	Get(key string, v interface{}) error
	Set(key string, v interface{}, ttl time.Duration) error
	Has(key string) bool
	Del(key string) error
	// Take takes the result from cache first, if not found,
	// call query to get a value(such as query database) and set cache using c.expiry, then return the result.
	Take(v interface{}, key string, ttl time.Duration, query func() (interface{}, error)) error
	// IsNotFound detect the error weather not found from cache
	IsNotFound(err error) bool
}
