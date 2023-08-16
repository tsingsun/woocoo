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

func NewClient(cfg *conf.Configuration) (*Client, error) {
	v := &Client{}
	if err := v.Apply(cfg); err != nil {
		return nil, err
	}
	return v, nil
}

// NewBuiltIn return a Client through application default
func NewBuiltIn() *Client {
	c, err := NewClient(conf.Global().Sub("store.redis"))
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Client) Close() error {
	if c.UniversalClient == nil {
		return nil
	}
	return c.UniversalClient.Close()
}

// Apply implements the conf.Configurable interface
func (c *Client) Apply(cfg *conf.Configuration) error {
	opts := redis.UniversalOptions{}
	err := cfg.Unmarshal(&opts)
	if err != nil {
		return err
	}
	c.redisOptions = &opts
	c.UniversalClient = redis.NewUniversalClient(&opts)
	return nil
}
