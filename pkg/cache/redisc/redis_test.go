package redisc_test

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/cache/redisc"
	"github.com/tsingsun/woocoo/pkg/conf"
	"testing"
	"time"
)

func initStandaloneCache(t *testing.T) (*redisc.Redisc, *miniredis.Miniredis) {
	cfgstr := `
cache:
  redis:
    driverName: 
    #    type: cluster
    #    addrs:
    #      - 127.0.0.1:6379
    type: standalone
    addr: 127.0.0.1:6379
    db: 1
    local:
      size: 1000
      ttl: 60
`
	cfg := conf.NewFromBytes([]byte(cfgstr)).Load()
	mr := miniredis.RunT(t)
	cfg.Parser().Set("cache.redis.addr", mr.Addr())
	cfg.Parser().Set("cache.redis.driverName", mr.Addr())
	cache := redisc.NewBuiltIn()
	return cache, mr
}

func TestNew(t *testing.T) {
	type args struct {
		cfg *conf.Configuration
		cli redis.Cmdable
	}
	tests := []struct {
		name string
		args args
		want func(*redisc.Redisc, *testing.T)
	}{
		{
			name: "local",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
					"local": map[string]interface{}{
						"size": 1000,
						"ttl":  "60s",
					},
				})),
			},
			want: func(r *redisc.Redisc, t *testing.T) {
				assert.Nil(t, r.RedisClient())
				assert.True(t, r.LocalCacheEnabled())
			},
		},
		{
			name: "local with redis",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"local": map[string]interface{}{
						"size": 1000,
						"ttl":  "60s",
					},
				})),
				cli: redis.NewClient(&redis.Options{}),
			},
			want: func(r *redisc.Redisc, t *testing.T) {
				assert.NotNil(t, r.RedisClient())
				assert.True(t, r.LocalCacheEnabled())
			},
		},
		{
			name: "redis and local",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"local": map[string]interface{}{
						"size": 1000,
						"ttl":  "60s",
					},
					"type": "standalone",
					"addr": "127.0.0.1:6379",
					"db":   1,
				})),
			},
			want: func(r *redisc.Redisc, t *testing.T) {
				assert.NotNil(t, r.RedisClient())
				assert.True(t, r.LocalCacheEnabled())
			},
		},
		{
			name: "standalone",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"local": map[string]interface{}{
						"size": 1000,
						"ttl":  "60s",
					},
					"type": "standalone",
					"addr": "127.0.0.1:6379",
					"db":   1,
				})),
			},
			want: func(r *redisc.Redisc, t *testing.T) {
				assert.NotNil(t, r.RedisClient())
				assert.True(t, r.LocalCacheEnabled())
			},
		},
		{
			name: "cluster",
			args: args{
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]interface{}{
					"local": map[string]interface{}{
						"size": 1000,
						"ttl":  "60s",
					},
					"type": "cluster",
					"addr": []string{"127.0.0.1:6379"},
					"db":   1,
				})),
			},
			want: func(r *redisc.Redisc, t *testing.T) {
				assert.NotNil(t, r.RedisClient())
				assert.True(t, r.LocalCacheEnabled())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redisc.New(tt.args.cfg, tt.args.cli)
			tt.want(got, t)
		})
	}
}

func TestCache_Take(t *testing.T) {
	cache, mr := initStandaloneCache(t)
	defer mr.Close()
	got := "string"
	want := "abc"
	err := cache.Take(context.Background(), &got, "string", time.Minute, func() (interface{}, error) {
		return want, nil
	})
	assert.NoError(t, err)
	if got != want {
		t.Errorf("got %v,but want %v", got, want)
	}
}

func TestCache_Set(t *testing.T) {
	cache, mr := initStandaloneCache(t)
	defer mr.Close()
	got := "string"
	if err := cache.Set(context.Background(), "string", got, time.Hour); err != nil {
		t.Error(err)
	}
	var want string
	if err := cache.Get(context.Background(), "string", &want); err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("got %v,but want %v", got, want)
	}

}
