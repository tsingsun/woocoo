package redis

import (
	"context"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/tsingsun/woocoo/pkg/conf"
	"time"
)

//Cache implement github.com/tsingsun/woocoo/cache/Cache
type Cache struct {
	client *cache.Cache
}

func (c *Cache) Apply(cnf *conf.Config, path string) {
	var local *cache.TinyLFU
	if cnf.IsSet("cache.local") {
		local = cache.NewTinyLFU(cnf.Int("cache.local.size"), time.Second*time.Duration(cnf.Int("cache.local.ttl")))
	}
	cc, err := cnf.Operator().Sub(path)
	if err != nil {
		panic(err)
	}
	clientType := cc.Get("type")
	switch clientType {
	case "cluster":
		opts := &redis.ClusterOptions{}
		err = cc.UnmarshalByJson("redis", opts)
		cl := redis.NewClusterClient(opts)
		c.client = cache.New(&cache.Options{
			Redis:      cl,
			LocalCache: local,
		})
	case "ring":
		opts := &redis.RingOptions{}
		err = cc.UnmarshalByJson("redis", opts)
		cl := redis.NewRing(opts)
		c.client = cache.New(&cache.Options{
			Redis:      cl,
			LocalCache: local,
		})
	case "standalone":
	default:
		opts := &redis.Options{}
		err = cc.UnmarshalByJson("redis", opts)
		cl := redis.NewClient(opts)
		c.client = cache.New(&cache.Options{
			Redis:      cl,
			LocalCache: local,
		})
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
