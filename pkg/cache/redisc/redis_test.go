package redisc

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	type args struct {
		cfg  *conf.Configuration
		opts []Option
	}
	tests := []struct {
		name string
		args args
		want func(*Redisc, *testing.T)
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
			want: func(r *Redisc, t *testing.T) {
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
				opts: []Option{
					WithRedisClient(redis.NewClient(&redis.Options{})),
				},
			},
			want: func(r *Redisc, t *testing.T) {
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
			want: func(r *Redisc, t *testing.T) {
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
			want: func(r *Redisc, t *testing.T) {
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
			want: func(r *Redisc, t *testing.T) {
				assert.NotNil(t, r.RedisClient())
				assert.True(t, r.LocalCacheEnabled())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.cfg, tt.args.opts...)
			tt.want(got, t)
		})
	}
}

func initStandaloneRedisc(t *testing.T) (*Redisc, *miniredis.Miniredis) {
	cfgstr := `
redis: 
  type: standalone
  addr: 127.0.0.1:6379
  db: 1
  local:
    size: 1000
    ttl: 60s
`
	cfg := conf.NewFromBytes([]byte(cfgstr)).Load()
	mr := miniredis.RunT(t)
	mr.Select(cfg.Int("redis.db"))
	cfg.Parser().Set("redis.addr", mr.Addr())
	cfg.Parser().Set("redis.driverName", mr.Addr())
	redisc := New(cfg.Sub("redis"))
	return redisc, mr
}

func TestRedisc(t *testing.T) {
	tests := []struct {
		name string
		do   func()
	}{
		{
			name: "set no expire",
			do: func() {
				rdc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rdc.Set(context.Background(), "key", "value", 0))
				rdb.FastForward(time.Hour * 24)
				assert.True(t, rdc.Has(context.Background(), "key"))
				want := ""
				assert.NoError(t, rdc.Get(context.Background(), "key", &want))
				assert.Equal(t, "value", want)
			},
		},
		{
			name: "set expire",
			do: func() {
				rdc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rdc.Set(context.Background(), "key", "value", time.Second))
				rdb.FastForward(time.Hour)
				time.Sleep(time.Second * 2)
				want := ""
				assert.False(t, rdc.Has(context.Background(), "key"))
				assert.Error(t, rdc.Get(context.Background(), "key", &want))
			},
		},
		{
			name: "take",
			do: func() {
				rdc, rdb := initStandaloneRedisc(t)
				want := ""
				err := rdc.Take(context.Background(), &want, "key", time.Second, func() (any, error) {
					return "123", nil
				})
				assert.NoError(t, err)
				assert.Equal(t, "123", want)
				assert.True(t, rdb.Exists("key"))
				assert.NoError(t, rdc.Del(context.Background(), "key"))
				assert.False(t, rdb.Exists("key"))
			},
		},
		{
			name: "take with expire",
			do: func() {
				rdc, rdb := initStandaloneRedisc(t)
				want := ""
				err := rdc.Take(context.Background(), &want, "key", time.Second, func() (any, error) {
					return "123", nil
				})
				assert.NoError(t, err)
				time.Sleep(time.Second * 2)
				rdb.FastForward(time.Hour)
				assert.False(t, rdc.Has(context.Background(), "key"))
				assert.False(t, rdb.Exists("key"))
			},
		},
		{
			name: "not found error",
			do: func() {
				rdc, _ := initStandaloneRedisc(t)
				err := rdc.Get(context.Background(), "key", nil)
				assert.True(t, rdc.IsNotFound(err))
			},
		},
		{
			name: "cache interface-int",
			do: func() {
				rdc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rdc.Set(context.Background(), "key", 1, 0))
				rdb.FastForward(time.Hour * 24)
				assert.True(t, rdc.Has(context.Background(), "key"))
				got := 0
				assert.NoError(t, rdc.Get(context.Background(), "key", &got))
				assert.Equal(t, 1, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
		})
	}
}

func TestNewBuiltIn(t *testing.T) {
	tests := []struct {
		name    string
		cnf     *conf.Configuration
		wantErr bool
	}{
		{
			name: "builtin",
			cnf: conf.NewFromStringMap(map[string]any{
				"cache": map[string]any{
					"redis": map[string]any{
						"type": "standalone",
						"addr": "127.0.0.1:6379",
						"db":   1,
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cnf.AsGlobal()
			if tt.wantErr {
				assert.Panics(t, func() {
					NewBuiltIn()
				})
				return
			}
			assert.NotNil(t, NewBuiltIn())
		})
	}
}
