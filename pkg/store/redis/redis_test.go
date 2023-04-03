package redis

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"testing"
)

func TestNewClient(t *testing.T) {
	b := `
store:
  redis1: 
    addrs:
      - 127.0.0.1:6379
      - 127.0.0.1:6380
  redis: 
    addrs: 
      - 127.0.0.1:6379
    db: 1
    dialTimeout: 5s
  redisr: 
    masterName: "mymaster"
    addrs: 
      - localhost:7000
`
	tests := []struct {
		name      string
		cfg       *conf.Configuration
		newFunc   func() *Client
		wantPanic bool
	}{
		{
			name: "builtin",
			newFunc: func() *Client {
				rds := miniredis.RunT(t)
				cfg := conf.NewFromBytes([]byte(b)).Load()
				cfg.Parser().Set("store.redis.addr", rds.Addr())
				cfg.AsGlobal()
				return NewBuiltIn()
			},
		},
		{
			name: "standalone",
			newFunc: func() *Client {
				rds := miniredis.RunT(t)
				cfg := conf.NewFromBytes([]byte(b)).Load().Sub("store.redis")
				cfg.Parser().Set("addr", rds.Addr())
				cli := NewClient(cfg)
				assert.Equal(t, cli.redisOptions.(*redis.UniversalOptions).DialTimeout, cfg.Duration("dialTimeout"))
				return cli
			},
		},
		{
			name: "cluster",
			newFunc: func() *Client {
				rds := miniredis.RunT(t)
				rds1 := miniredis.RunT(t)
				cfg := conf.NewFromBytes([]byte(b)).Load().Sub("store.redis1")
				cfg.Parser().Set("addrs", []string{rds.Addr(), rds1.Addr()})
				return NewClient(cfg)
			},
		},
		{
			name: "fail over",
			newFunc: func() *Client {
				rds := miniredis.RunT(t)
				cfg := conf.NewFromBytes([]byte(b)).Load().Sub("store.redisr")
				cfg.Parser().Set("addrs", []string{rds.Addr()})
				return NewClient(cfg)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.name == "builtin" {
				client = tt.newFunc()
			} else {
				if tt.wantPanic {
					assert.Panics(t, func() {
						tt.newFunc()
					})
					return
				}

				client = tt.newFunc()
			}
			client.Ping(context.Background())
			assert.NoError(t, client.Close())
		})
	}
}
