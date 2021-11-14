package redis

import (
	"context"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	manager "github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"time"
)

//Cache implement github.com/tsingsun/woocoo/cache/Cache
type Cache struct {
	client *cache.Cache
}

// Apply
// rCfg is the root configuration
func (c *Cache) Apply(rCfg *conf.Configuration, path string) {
	var local *cache.TinyLFU
	var err error
	cnf := rCfg.Sub(path)
	if cnf.IsSet("local") {
		local = cache.NewTinyLFU(cnf.Int("local.size"), time.Second*time.Duration(cnf.Int("local.ttl")))
	}
	clientType := cnf.Get("redis.type")
	switch clientType {
	case "cluster":
		opts := &redis.ClusterOptions{}
		err = cnf.Parser().Unmarshal("redis", opts)
		cl := redis.NewClusterClient(opts)
		c.client = cache.New(&cache.Options{
			Redis:      cl,
			LocalCache: local,
		})
	case "ring":
		opts := &redis.RingOptions{}
		err = cnf.Parser().Unmarshal("redis", opts)
		cl := redis.NewRing(opts)
		c.client = cache.New(&cache.Options{
			Redis:      cl,
			LocalCache: local,
		})
	case "standalone":
		fallthrough
	default:
		opts := &redis.Options{}
		err = cnf.Parser().Unmarshal("redis", opts)
		cl := redis.NewClient(opts)
		c.client = cache.New(&cache.Options{
			Redis:      cl,
			LocalCache: local,
		})
	}
	if err = manager.RegisterCache(path, c); err != nil {
		panic(err)
	}
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
