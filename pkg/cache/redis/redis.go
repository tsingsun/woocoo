package redis

import (
	"context"
	"github.com/go-redis/cache/v8"
	manager "github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	store "github.com/tsingsun/woocoo/pkg/store/redis"
	"time"
)

//Cache implement github.com/tsingsun/woocoo/cache/Cache
type Cache struct {
	client *cache.Cache
}

func NewBuiltIn() *Cache {
	c := &Cache{}
	cfg := conf.Global().Sub("cache.redis")
	c.Apply(cfg)
	var driverName = "redis"
	if cfg.IsSet("driverName") {
		driverName = cfg.String("driverName")
	}
	if err := manager.RegisterCache(driverName, c); err != nil {
		panic(err)
	}
	return c
}

// Apply conf.configurable
func (c *Cache) Apply(cfg *conf.Configuration) {
	var (
		opts   cache.Options
		local  *cache.TinyLFU
		remote *store.Client
		err    error
	)
	if cfg.IsSet("local") {
		local = cache.NewTinyLFU(cfg.Int("local.size"), cfg.Duration("local.ttl"))
		opts.LocalCache = local
	}
	if cfg.IsSet("type") {
		remote, err = store.NewClient()
		if err != nil {
			panic(err)
		}
		remote.Apply(cfg)
		opts.Redis = remote
	}
	c.client = cache.New(&opts)
}

func (c *Cache) Get(key string, v interface{}) error {
	return c.client.Get(context.Background(), key, v)
}

func (c *Cache) Set(key string, v interface{}, ttl time.Duration) error {
	return c.client.Set(&cache.Item{
		Key:   key,
		Value: v,
		TTL:   ttl,
	})
}

func (c *Cache) Has(key string) bool {
	return c.client.Exists(context.Background(), key)
}

func (c *Cache) Del(key string) error {
	return c.client.Delete(context.Background(), key)
}

func (c *Cache) Take(v interface{}, key string, ttl time.Duration, query func() (interface{}, error)) error {
	item := &cache.Item{
		Key:   key,
		Value: v,
		TTL:   ttl,
		Do: func(item *cache.Item) (interface{}, error) {
			return query()
		},
	}
	return c.client.Once(item)
}

func (c *Cache) IsNotFound(err error) bool {
	return err == cache.ErrCacheMiss
}
