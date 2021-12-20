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

// NewGhostClient return a Client through application default
func NewGhostClient() *Client {
	v := &Client{}
	v.Apply(conf.Global(), "store.redis")
	return v
}

func (c *Client) Close() error {
	if c.Cmdable == nil {
		return nil
	}
	return c.closeFunc()
}

func (c *Client) Apply(cfg *conf.Configuration, path string) {
	var err error
	cnf := cfg.Sub(path)
	c.ClientType = cnf.String("type")
	switch c.ClientType {
	case "cluster":
		opts := &redis.ClusterOptions{}
		err = cnf.Parser().Unmarshal("", opts)
		cl := redis.NewClusterClient(opts)
		c.closeFunc = cl.Close
		c.option = opts
		c.Cmdable = cl
	case "ring":
		opts := &redis.RingOptions{}
		err = cnf.Parser().Unmarshal("", opts)
		cl := redis.NewRing(opts)
		c.closeFunc = cl.Close
		c.option = opts
		c.Cmdable = cl
	case "standalone":
		fallthrough
	default:
		opts := &redis.Options{}
		err = cnf.Parser().Unmarshal("", opts)
		cl := redis.NewClient(opts)
		c.closeFunc = cl.Close
		c.option = opts
		c.Cmdable = cl
	}
	if err != nil {
		panic(err)
	}

}
