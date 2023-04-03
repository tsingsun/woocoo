package redisc

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	manager "github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	store "github.com/tsingsun/woocoo/pkg/store/redis"
	"time"
)

type (
	// Redisc implement github.com/tsingsun/woocoo/cache/Cache
	//
	// if you want to register to cache manager, set a `driverName` in configuration
	Redisc struct {
		operator   *Cache
		client     redis.UniversalClient
		driverName string
	}

	Option func(*Redisc)
)

func WithRedisClient(cli redis.UniversalClient) Option {
	return func(redisc *Redisc) {
		redisc.client = cli
	}
}

func New(cfg *conf.Configuration, opts ...Option) *Redisc {
	c := &Redisc{
		driverName: "redis",
	}
	for _, opt := range opts {
		opt(c)
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
	c := New(conf.Global().Sub("cache.redis"))
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
		if cfg.IsSet("addrs") {
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
func (c *Redisc) Get(ctx context.Context, key string, v any) error {
	return c.operator.Get(ctx, key, v)
}

// Set sets the value associated with the given key.
// if ttl < 0 ,will not save to redis,but save to local cache if enabled
func (c *Redisc) Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	return c.operator.Set(&Item{
		Ctx:   ctx,
		Key:   key,
		Value: v,
		TTL:   ttl,
	})
}

// Has returns true if the given key exists.
func (c *Redisc) Has(ctx context.Context, key string) bool {
	return c.operator.Exists(ctx, key)
}

// Del deletes the given key.
func (c *Redisc) Del(ctx context.Context, key string) error {
	return c.operator.Delete(ctx, key)
}

// Take returns the value associated with the given key. It Uses flight control to avoid concurrent requests.
func (c *Redisc) Take(ctx context.Context, v any, key string, ttl time.Duration, query func() (any, error)) error {
	item := &Item{
		Ctx:   ctx,
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
