package cache

import (
	"context"
	"fmt"
	"time"
)

var (
	_manager          = newManager()
	DefaultDriverName = "redis"
	_defaultDriver    Cache
)

type manager struct {
	drivers map[string]Cache
}

func newManager() *manager {
	return &manager{
		drivers: make(map[string]Cache),
	}
}

func SetDefault(driver string) error {
	DefaultDriverName = driver
	_defaultDriver = _manager.drivers[DefaultDriverName]
	return nil
}

func RegisterCache(name string, cache Cache) error {
	if _, ok := _manager.drivers[name]; ok {
		return fmt.Errorf("driver already registered for name %q", name)
	}
	_manager.drivers[name] = cache
	if len(_manager.drivers) == 1 {
		return SetDefault(name)
	}
	return nil
}

// GetCache return a Cache driver,if it has multi Cache Driver
func GetCache(driver string) Cache {
	return _manager.drivers[driver]
}

func Get(ctx context.Context, key string, v any) error {
	return _defaultDriver.Get(ctx, key, v)
}

func Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	return _defaultDriver.Set(ctx, key, v, ttl)
}

func Has(ctx context.Context, key string) bool {
	return _defaultDriver.Has(ctx, key)
}

func Del(ctx context.Context, key string) error {
	return _defaultDriver.Del(ctx, key)
}

func Take(ctx context.Context, v any, key string, ttl time.Duration, query func() (any, error)) error {
	return _defaultDriver.Take(ctx, v, key, ttl, query)
}

func IsNotFound(err error) bool {
	return _defaultDriver.IsNotFound(err)
}
