package redisc

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	manager "github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	store "github.com/tsingsun/woocoo/pkg/store/redis"
	"time"
)

// Redisc implement github.com/tsingsun/woocoo/cache/Cache
//
// if you want to register to cache manager, set a `driverName` in configuration
type Redisc struct {
	operator   *Cache
	client     redis.Cmdable
	driverName string
}

func New(cfg *conf.Configuration, cli redis.Cmdable) *Redisc {
	c := &Redisc{
		client:     cli,
		driverName: "redis",
	}
	c.Apply(cfg)
	if cfg.IsSet("driverName") && cfg.String("driverName") != "" {
		c.driverName = cfg.String("driverName")
		if err := c.Register(); err != nil {
			panic(err)
		}
	}
	return c
}

func NewBuiltIn() *Redisc {
	c := New(conf.Global().Sub("cache.redis"), nil)
	return c
}

func (c *Redisc) Register() error {
	return manager.RegisterCache(c.driverName, c)
}

// Apply conf.configurable
func (c *Redisc) Apply(cfg *conf.Configuration) {
	var (
		opts  Options
		local *TinyLFU
	)
	if c.client != nil {
		opts.Redis = c.client
	}
	if cfg.Development {
		opts.StatsEnabled = true
	}
	if cfg.IsSet("local") {
		opts.LocalCacheTTL = cfg.Duration("local.ttl")
		local = NewTinyLFU(cfg.Int("local.size"), opts.LocalCacheTTL)
		opts.LocalCache = local
	}
	if c.client == nil {
		if cfg.IsSet("type") {
			remote := store.NewClient(cfg)
			opts.Redis = remote
			c.client = remote
		}
	}
	if opts.Redis == nil && opts.LocalCache == nil {
		panic("redis cache must have a redis client or local cache")
	}
	c.operator = NewCache(&opts)
}

// Get returns the value associated with the given key.
func (c *Redisc) Get(key string, v any) error {
	return c.operator.Get(context.Background(), key, v)
}

// Set sets the value associated with the given key.
func (c *Redisc) Set(key string, v any, ttl time.Duration) error {
	return c.operator.Set(&Item{
		Key:   key,
		Value: v,
		TTL:   ttl,
	})
}

// Has returns true if the given key exists.
func (c *Redisc) Has(key string) bool {
	return c.operator.Exists(context.Background(), key)
}

// Del deletes the given key.
func (c *Redisc) Del(key string) error {
	return c.operator.Delete(context.Background(), key)
}

// Take returns the value associated with the given key.
func (c *Redisc) Take(v any, key string, ttl time.Duration, query func() (any, error)) error {
	item := &Item{
		Key:   key,
		Value: v,
		TTL:   ttl,
		Do: func(item *Item) (interface{}, error) {
			return query()
		},
	}
	return c.operator.Take(item)
}

// IsNotFound returns true if the error is cache.ErrCacheMiss.
func (c *Redisc) IsNotFound(err error) bool {
	return errors.Is(err, ErrCacheMiss)
}

// Operator returns the underlying Redisc.
func (c *Redisc) Operator() *Cache {
	return c.operator
}

// RedisClient returns the underlying redis client.
func (c *Redisc) RedisClient() redis.Cmdable {
	if c.operator.opt.Redis == nil {
		return nil
	}
	return c.operator.opt.Redis.(redis.Cmdable)
}

// LocalCacheEnabled returns true if local cache is enabled.
func (c *Redisc) LocalCacheEnabled() bool {
	return c.operator.opt.LocalCache != nil
}
