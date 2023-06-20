package cache

import (
	"context"
	"fmt"
)

var (
	_manager       = newManager()
	_defaultDriver Cache
)

type manager struct {
	drivers map[string]Cache
}

func newManager() *manager {
	return &manager{
		drivers: make(map[string]Cache),
	}
}

// SetDefault sets the default driver to use the static functions.
func SetDefault(driver string) error {
	_defaultDriver = _manager.drivers[driver]
	return nil
}

// RegisterCache registers a cache driver.
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

func Get(ctx context.Context, key string, v any, opts ...Option) error {
	return _defaultDriver.Get(ctx, key, v, opts...)
}

func Set(ctx context.Context, key string, v any, opts ...Option) error {
	return _defaultDriver.Set(ctx, key, v, opts...)
}

func Has(ctx context.Context, key string) bool {
	return _defaultDriver.Has(ctx, key)
}

func Del(ctx context.Context, key string) error {
	return _defaultDriver.Del(ctx, key)
}

func IsNotFound(err error) bool {
	return _defaultDriver.IsNotFound(err)
}
