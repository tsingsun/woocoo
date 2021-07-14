package cache

import (
	"time"
)

type Cache interface {
	Get(key string, v interface{}) error
	Set(key string, v interface{}, ttl time.Duration) error
	Has(key string) bool
	Del(key string) error
	Take(v interface{}, key string, ttl time.Duration, query func() (interface{}, error)) error
	IsNotFound(err error) bool
}
