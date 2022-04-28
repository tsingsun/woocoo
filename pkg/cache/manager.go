package cache

import (
	"fmt"
	"time"
)

var (
	_manager           = newManager()
	_defaultDriverName = "redis"
	_defaultDriver     Cache
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
	_defaultDriverName = driver
	_defaultDriver = _manager.drivers[_defaultDriverName]
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

func Get(key string, v interface{}) error {
	return _defaultDriver.Get(key, v)
}

func Set(key string, v interface{}, ttl time.Duration) error {
	return _defaultDriver.Set(key, v, ttl)
}

func Has(key string) bool {
	return _defaultDriver.Has(key)
}

func Del(key string) error {
	return _defaultDriver.Del(key)
}

func Take(v interface{}, key string, ttl time.Duration, query func() (interface{}, error)) error {
	return _defaultDriver.Take(v, key, ttl, query)
}

func IsNotFound(err error) bool {
	return _defaultDriver.IsNotFound(err)
}
