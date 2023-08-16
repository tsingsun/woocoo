package redisc

import (
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/testco/wctest"
	"testing"
	"time"
)

func initStandaloneRedisc(t *testing.T) (*Redisc, *miniredis.Miniredis) {
	cfgstr := `
redis: 
  stats: true
  addrs: 
    - 127.0.0.1:6379
  db: 1
  local:
    size: 1000
    ttl: 60s
`
	cfg := conf.NewFromBytes([]byte(cfgstr)).Load()
	mr := miniredis.RunT(t)
	mr.Select(cfg.Int("redis.db"))
	cfg.Parser().Set("redis.addrs", []string{mr.Addr()})
	redisc, err := New(cfg.Sub("redis"))
	require.NoError(t, err)
	assert.NotNil(t, redisc.Stats())
	return redisc, mr
}

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
					"local": map[string]any{
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
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
					"driverName": "redis",
					"local": map[string]any{
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
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
					"local": map[string]any{
						"size": 1000,
						"ttl":  "60s",
					},
					"addrs": []string{"127.0.0.1:6379"},
					"db":    1,
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
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
					"local": map[string]any{
						"size": 1000,
						"ttl":  "60s",
					},
					"addrs": []string{"127.0.0.1:6379"},
					"db":    1,
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
				cfg: conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
					"local": map[string]any{
						"size": 1000,
						"ttl":  "60s",
					},
					"addrs": []string{"127.0.0.1:6379"},
					"db":    1,
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
			got, err := New(tt.args.cfg, tt.args.opts...)
			require.NoError(t, err)
			tt.want(got, t)
		})
	}
}

func TestRedisc(t *testing.T) {
	tests := []struct {
		name string
		do   func()
	}{
		{
			name: "use default ttl",
			do: func() {
				rd, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rd.Set(context.Background(), "key", "value"))
				rdb.FastForward(time.Hour * 24) // default test ttl is 60s
				assert.True(t, rd.Has(context.Background(), "key"))
				want := ""
				assert.NoError(t, rd.Get(context.Background(), "key", &want))
				assert.Equal(t, "value", want)
			},
		},
		{
			name: "set expire",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				key := "expire"
				assert.NoError(t, rc.Set(context.Background(), key, "value", cache.WithTTL(time.Second)))
				rdb.FastForward(time.Hour)
				time.Sleep(time.Second * 2) // make local cache expire
				want := ""
				assert.False(t, rc.Has(context.Background(), key))
				assert.Error(t, rc.Get(context.Background(), key, &want))
			},
		},
		{
			name: "take",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				want := ""
				err := rc.Get(context.Background(), "key", &want,
					cache.WithTTL(time.Second), cache.WithGetter(func(ctx context.Context, key string) (any, error) {
						return "123", nil
					}))
				assert.NoError(t, err)
				assert.Equal(t, "123", want)
				assert.True(t, rdb.Exists("key"))
				assert.NoError(t, rc.Del(context.Background(), "key"))
				assert.False(t, rdb.Exists("key"))
			},
		},
		{
			name: "take with expire",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				want := ""
				err := rc.Get(context.Background(), "key", &want, cache.WithTTL(time.Second),
					cache.WithGetter(func(ctx context.Context, key string) (any, error) {
						return "123", nil
					}))
				assert.NoError(t, err)
				time.Sleep(time.Second * 2)
				rdb.FastForward(time.Hour)
				assert.False(t, rc.Has(context.Background(), "key"))
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
				assert.NoError(t, rdc.Set(context.Background(), "key", 1))
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

func TestCache_Op(t *testing.T) {
	tests := []struct {
		name string
		do   func()
	}{
		{
			name: "get set item value nil",
			do: func() {
				rc, rds := initStandaloneRedisc(t)
				assert.NoError(t, rc.Set(context.Background(), "key", nil, cache.WithTTL(time.Millisecond*100)))
				assert.NoError(t, rc.Get(context.Background(), "key", nil))
				assert.True(t, rc.Has(context.Background(), "key"))
				assert.True(t, rds.Exists("key"))
			},
		},
		{
			name: "deletes key",
			do: func() {
				rc, rds := initStandaloneRedisc(t)
				assert.NoError(t, rc.Set(context.Background(), "key", nil, cache.WithTTL(time.Hour)))
				assert.True(t, rc.Has(context.Background(), "key"))
				assert.NoError(t, rc.Del(context.Background(), "key"))
				assert.False(t, rc.Has(context.Background(), "key"))
				assert.False(t, rds.Exists("key"))
			},
		},
		{
			name: "gets and sets data",
			do: func() {
				rc, rds := initStandaloneRedisc(t)
				v := struct {
					Name  string
					Count int
				}{
					Name:  "test",
					Count: 1,
				}

				assert.NoError(t, rc.Set(context.Background(), "key", v, cache.WithTTL(time.Hour)))
				var want struct {
					Name  string
					Count int
				}
				assert.NoError(t, rc.Get(context.Background(), "key", &want))
				assert.Equal(t, v, want)
				assert.True(t, rc.Has(context.Background(), "key"))
				assert.True(t, rds.Exists("key"))
			},
		},
		{
			name: "sets string as is",
			do: func() {
				rc, _ := initStandaloneRedisc(t)
				value := "str_value"
				assert.NoError(t, rc.Set(context.Background(), "key", value))
				var want string
				assert.NoError(t, rc.Get(context.Background(), "key", &want))
				assert.Equal(t, value, want)
			},
		},
		{
			name: "set xx can be used with Incr",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rc.Set(context.Background(), "key", "123",
					cache.WithTTL(time.Hour),
					cache.WithSetNX()),
				)
				v, err := rdb.Incr("key", 1)
				assert.NoError(t, err)
				assert.Equal(t, int(124), v)
				assert.Error(t, rc.Set(context.Background(), "key", "125",
					cache.WithTTL(time.Hour), cache.WithSetNX()),
				)
				v1, err := rdb.Get("key")
				assert.NoError(t, err)
				assert.Equal(t, "124", v1)

				assert.Error(t, rc.Set(context.Background(), "key1", "125",
					cache.WithTTL(time.Hour), cache.WithSetXX()),
				)
				assert.False(t, rdb.Exists("key1"))
				assert.NoError(t, rc.Set(context.Background(), "key", "125",
					cache.WithTTL(time.Hour), cache.WithSetXX()),
				)
				assert.True(t, rdb.Exists("key"))
			},
		},
		{
			name: "get and set by skip",
			do: func() {
				type CT struct {
					Name string
					Data []byte
				}
				rc, rdb := initStandaloneRedisc(t)
				v := CT{
					Name: "test",
					Data: testdata.FileBytes("gzip/benchmark.json"),
				}
				ctx := context.Background()
				assert.NoError(t, rc.Set(ctx, "key", v,
					cache.WithTTL(time.Hour), cache.WithSkip(cache.SkipCache)))
				assert.False(t, rc.Has(ctx, "key"))
				assert.False(t, rdb.Exists("key"))
				assert.NoError(t, rc.Del(ctx, "key"))

				assert.NoError(t, rc.Set(ctx, "key", v,
					cache.WithTTL(time.Hour), cache.WithSkip(cache.SkipLocal)))

				want := CT{}
				assert.NoError(t, rc.Get(ctx, "key", &want, cache.WithSkip(cache.SkipLocal)))
				assert.ErrorIs(t, rc.Get(ctx, "key", &want, cache.WithSkip(cache.SkipRemote)),
					cache.ErrCacheMiss)
				assert.EqualValues(t, rc.stats.Hits, uint64(1))
				assert.Equal(t, v, want)
				assert.True(t, rdb.Exists("key"))
				assert.NoError(t, rc.Del(ctx, "key"))
				assert.False(t, rdb.Exists("key"))

				rc.local = nil
				assert.NoError(t, rc.Set(ctx, "key", v,
					cache.WithTTL(time.Hour), cache.WithSkip(cache.SkipCache)))
				rdb.FastForward(time.Hour * 2)
				var want1 *CT
				assert.ErrorIs(t, rc.Get(ctx, "key", want1), cache.ErrCacheMiss)
				assert.Nil(t, want1)
				assert.False(t, rc.Has(ctx, "key"))

				require.NoError(t, rc.Get(ctx, "key", &want1, cache.WithSkip(cache.SkipCache),
					cache.WithGetter(func(ctx context.Context, key string) (any, error) {
						return CT{Name: "load from db"}, nil
					})))
				assert.Equal(t, &CT{Name: "load from db"}, want1, "SkipCache should skip cached value")
			},
		},
		{
			name: "get and set by skip redis",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				want := ""
				assert.NoError(t, rc.Set(context.Background(), "key", "123", cache.WithSkip(cache.SkipRemote)))
				assert.NoError(t, rc.Get(context.Background(), "key", &want, cache.WithSkip(cache.SkipRemote)))
				assert.Equal(t, "123", want)
				assert.EqualValues(t, rc.stats.Hits, 0)
				assert.ErrorIs(t, rc.Get(context.Background(), "key", &want, cache.WithSkip(cache.SkipLocal)), cache.ErrCacheMiss)
				assert.EqualValues(t, rc.stats.Hits, 0)
				assert.False(t, rdb.Exists("key"))

				rc.DeleteFromLocalCache("key")
				assert.False(t, rc.Has(context.Background(), "key"))
			},
		},
		{
			name: "getter",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				want := ""
				ctx := context.Background()
				assert.NoError(t, rc.Set(ctx, "key", "123", cache.WithTTL(time.Hour), cache.WithGetter(
					func(ctx context.Context, key string) (any, error) {
						return "123", nil
					})))
				assert.NoError(t, rc.Get(ctx, "key", &want, cache.WithSkip(0)))
				assert.Equal(t, "123", want)
				assert.EqualValues(t, rc.stats.Hits, 0)
				v, err := rdb.Get("key")
				require.NoError(t, err)
				assert.Equal(t, "123", v)
				assert.ErrorContains(t, rc.Get(ctx, "noexist", &want, cache.WithGetter(func(ctx context.Context, key string) (any, error) {
					return "", fmt.Errorf("key %s not found", key)
				})), "key noexist not found")
			},
		},
		{
			name: "marshal and unmarshal error",
			do: func() {
				type TT struct {
					Name string
				}
				v := &TT{
					Name: "test",
				}
				rc, rdb := initStandaloneRedisc(t)
				var want *TT
				ctx := context.Background()
				assert.Error(t, rc.Get(ctx, "key", v, cache.WithGroup()))
				assert.NoError(t, rc.Set(ctx, "key", v), "marshal error when once")

				assert.NoError(t, rc.Get(ctx, "key", &want))
				assert.Equal(t, v, want)

				err := rc.Group(ctx, "key1", &want, &cache.Options{Getter: func(ctx context.Context, key string) (any, error) {
					return "no TT", nil
				}})
				assert.ErrorContains(t, err, "unknown compression method", "local has error data")
				rc.CleanLocalCache()
				rc.local = nil
				require.NoError(t, rdb.Set("key", "123"))
				err = rc.Group(ctx, "key", &want, &cache.Options{})
				assert.ErrorContains(t, err, "unknown compression method")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
		})
	}
}

func TestCache_Once(t *testing.T) {
	tests := []struct {
		name string
		do   func()
	}{
		{
			name: "single",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rc.Set(context.Background(), "key", "123", cache.WithTTL(time.Hour), cache.WithSkip(cache.SkipLocal)))
				err := wctest.RunWait(t, time.Second*2, func() error {
					want := ""
					assert.NoError(t, rc.Get(context.Background(), "key", &want, cache.WithGroup(), cache.WithSkip(cache.SkipLocal)))
					assert.Equal(t, "123", want)
					return nil
				}, func() error {
					want := ""
					assert.NoError(t, rc.Get(context.Background(), "key",
						&want, cache.WithGroup(), cache.WithSkip(cache.SkipLocal)))
					assert.Equal(t, "123", want)
					return nil
				})
				assert.NoError(t, err)
				assert.LessOrEqual(t, int(rc.stats.Hits), 2)
				assert.True(t, rdb.Exists("key"))
			},
		},
		{
			name: "take",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rc.Set(context.Background(), "key", "123", cache.WithTTL(time.Hour), cache.WithSkip(cache.SkipLocal)))
				err := wctest.RunWait(t, time.Second, func() error {
					want := ""
					assert.NoError(t, rc.Get(context.Background(), "key", &want, cache.WithGroup(), cache.WithSkip(cache.SkipLocal)))
					assert.Equal(t, "123", want)
					return nil
				}, func() error {
					want := ""
					assert.NoError(t, rc.Get(context.Background(), "key", &want, cache.WithGroup(), cache.WithSkip(cache.SkipLocal)))
					assert.Equal(t, "123", want)
					return nil
				})
				assert.NoError(t, err)
				assert.EqualValues(t, rc.stats.Hits, uint64(1))
				assert.True(t, rdb.Exists("key"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
		})
	}
}

func TestCache_CleanLocalCache(t *testing.T) {
	tests := []struct {
		name string
		do   func()
	}{
		{
			name: "clean local cache",
			do: func() {
				rc, rdb := initStandaloneRedisc(t)
				assert.NoError(t, rc.Set(context.Background(), "key", "123", cache.WithTTL(time.Hour)))
				rc.CleanLocalCache()
				assert.True(t, rdb.Exists("key"))
				assert.True(t, rc.Has(context.Background(), "key"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
		})
	}
}
