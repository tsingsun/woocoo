package cache

import (
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

func Get(key string, v any) error {
	return _defaultDriver.Get(key, v)
}

func Set(key string, v any, ttl time.Duration) error {
	return _defaultDriver.Set(key, v, ttl)
}

func Has(key string) bool {
	return _defaultDriver.Has(key)
}

func Del(key string) error {
	return _defaultDriver.Del(key)
}

func Take(v any, key string, ttl time.Duration, query func() (any, error)) error {
	return _defaultDriver.Take(v, key, ttl, query)
}

func IsNotFound(err error) bool {
	return _defaultDriver.IsNotFound(err)
}
