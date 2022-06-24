package redis

import (
	"github.com/go-redis/redis/v8"
	"github.com/tsingsun/woocoo/pkg/conf"
)

type Client struct {
	ClientType string
	redis.Cmdable
	closeFunc func() error
	option    interface{}
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
	if c.Cmdable == nil {
		return nil
	}
	return c.closeFunc()
}

func (c *Client) Apply(cfg *conf.Configuration) {
	var err error
	c.ClientType = cfg.String("type")
	switch c.ClientType {
	case "cluster":
		opts := &redis.ClusterOptions{}
		err = cfg.Unmarshal(opts)
		cl := redis.NewClusterClient(opts)
		c.closeFunc = cl.Close
		c.option = opts
		c.Cmdable = cl
	case "ring":
		opts := &redis.RingOptions{}
		err = cfg.Unmarshal(opts)
		cl := redis.NewRing(opts)
		c.closeFunc = cl.Close
		c.option = opts
		c.Cmdable = cl
	case "standalone":
		fallthrough
	default:
		opts := &redis.Options{}
		err = cfg.Unmarshal(opts)
		cl := redis.NewClient(opts)
		c.closeFunc = cl.Close
		c.option = opts
		c.Cmdable = cl
	}
	if err != nil {
		panic(err)
	}

}
