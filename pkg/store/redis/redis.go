package redis

import (
	"github.com/redis/go-redis/v9"
	"github.com/tsingsun/woocoo/pkg/conf"
)

// Client is a redis client wrapper. Use redis.UniversalClient instead client/clusterClient since v9
type Client struct {
	redis.UniversalClient
	redisOptions any
}

func NewClient(cfg *conf.Configuration) *Client {
	v := &Client{}
	v.Apply(cfg)
	return v
}

// NewBuiltIn return a Client through application default
func NewBuiltIn() *Client {
	return NewClient(conf.Global().Sub("store.redis"))
}

func (c *Client) Close() error {
	if c.UniversalClient == nil {
		return nil
	}
	return c.UniversalClient.Close()
}

func (c *Client) Apply(cfg *conf.Configuration) {
	opts := redis.UniversalOptions{}
	err := cfg.Unmarshal(&opts)
	if err != nil {
		panic(err)
	}
	c.redisOptions = &opts
	c.UniversalClient = redis.NewUniversalClient(&opts)
}
