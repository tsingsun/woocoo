package redisc

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/internal/wctest"
	"testing"
	"time"
)

func TestSkipMode(t *testing.T) {
	type args struct {
		mode SkipMode
	}
	tests := []struct {
		name string
		f    SkipMode
		Func func(mode SkipMode) bool
		args args
		want bool
	}{
		{
			name: "in",
			f:    SkipLocal,
			Func: SkipLocal.Is,
			args: args{
				mode: SkipRedis,
			},
			want: false,
		},
		{
			name: "0in",
			f:    SkipMode(0),
			Func: SkipMode(0).Is,
			args: args{
				mode: SkipLocal,
			},
			want: false,
		},
		{
			name: "any",
			f:    SkipLocal,
			Func: SkipLocal.Is,
			args: args{
				mode: SkipMode(0),
			},
			want: false,
		},
		{
			name: "none",
			f:    SkipMode(0),
			Func: func(mode SkipMode) bool {
				return mode.Any()
			},
			args: args{
				mode: SkipMode(0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.Func(tt.args.mode), "Is(%v)", tt.args.mode)
		})
	}
}

func initStandaloneCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	rc, mr := initStandaloneRedisc(t)
	cache := rc.Operator()
	cache.opt.StatsEnabled = true
	return cache, mr
}

func TestCache_Set(t *testing.T) {
	tests := []struct {
		name string
		do   func()
	}{
		{
			name: "set nil,item never nil",
			do: func() {
				cache, _ := initStandaloneCache(t)
				assert.Panics(t, func() {
					_ = cache.Set(nil)
				})
			},
		},
		{
			name: "get set item value nil",
			do: func() {
				cache, rds := initStandaloneCache(t)
				item := &Item{
					Key: "key",
					TTL: time.Millisecond * 100,
				}
				assert.NoError(t, cache.Set(item))
				assert.Equal(t, item.ttl(), time.Hour)
				assert.NoError(t, cache.Get(context.Background(), "key", nil))
				assert.True(t, cache.Exists(context.Background(), "key"))
				assert.True(t, rds.Exists("key"))
			},
		},
		{
			name: "deletes key",
			do: func() {
				cache, rds := initStandaloneCache(t)
				item := &Item{
					Ctx: context.Background(),
					Key: "key",
					TTL: time.Hour,
				}
				assert.NoError(t, cache.Set(item))
				assert.True(t, cache.Exists(context.Background(), "key"))
				assert.NoError(t, cache.Delete(context.Background(), "key"))
				assert.False(t, cache.Exists(context.Background(), "key"))
				assert.False(t, rds.Exists("key"))
			},
		},
		{
			name: "gets and sets data",
			do: func() {
				cache, rds := initStandaloneCache(t)
				item := &Item{
					Ctx: context.Background(),
					Key: "key",
					TTL: time.Hour,
					Value: struct {
						Name  string
						Count int
					}{
						Name:  "test",
						Count: 1,
					},
				}
				assert.NoError(t, cache.Set(item))
				var want struct {
					Name  string
					Count int
				}
				assert.NoError(t, cache.Get(context.Background(), "key", &want))
				assert.Equal(t, item.Value, want)
				assert.True(t, cache.Exists(context.Background(), "key"))
				assert.True(t, rds.Exists("key"))
			},
		},
		{
			name: "Sets string as is",
			do: func() {
				cache, _ := initStandaloneCache(t)
				value := "str_value"
				item := &Item{
					Ctx:   context.Background(),
					Key:   "key",
					Value: value,
				}
				assert.NoError(t, cache.Set(item))
				var want string
				assert.NoError(t, cache.Get(context.Background(), "key", &want))
				assert.Equal(t, item.Value, want)
			},
		},
		{
			name: "set xx can be used with Incr",
			do: func() {
				cache, rdb := initStandaloneCache(t)
				item := &Item{
					Ctx:   context.Background(),
					Key:   "key",
					TTL:   time.Hour,
					Value: "123",
					SetNX: true,
				}
				assert.NoError(t, cache.Set(item))
				v, err := rdb.Incr("key", 1)
				assert.NoError(t, err)
				assert.Equal(t, int(124), v)
				item.Value = "125"
				assert.NoError(t, cache.Set(item))
				v1, err := rdb.Get("key")
				assert.NoError(t, err)
				assert.Equal(t, "124", v1)

				item.SetXX = true
				item.Key = "key1"
				assert.NoError(t, cache.Set(item))
				assert.False(t, rdb.Exists("key1"))
				item.Key = "key"
				assert.NoError(t, cache.Set(item))
				assert.True(t, rdb.Exists("key"))
			},
		},
		{
			name: "get and set by skip local",
			do: func() {
				cache, rdb := initStandaloneCache(t)
				item := &Item{
					Ctx:   context.Background(),
					Key:   "key",
					TTL:   time.Hour,
					Value: "123",
					Skip:  SkipAll,
				}
				assert.NoError(t, cache.Set(item))
				assert.False(t, cache.Exists(context.Background(), "key"))
				assert.False(t, rdb.Exists("key"))
				assert.NoError(t, cache.Delete(context.Background(), "key"))

				item.Skip = SkipLocal
				assert.NoError(t, cache.Set(item))
				// assert.True(t, cache.Exists(context.Background(), "key"))
				want := ""
				assert.NoError(t, cache.GetSkip(context.Background(), "key", &want, item.Skip))
				assert.ErrorIs(t, cache.GetSkip(context.Background(), "key", &want, SkipRedis), ErrCacheMiss)
				assert.EqualValues(t, cache.Stats().Hits, 1)
				assert.Equal(t, item.Value, want)
				assert.True(t, rdb.Exists("key"))
				assert.NoError(t, cache.Delete(context.Background(), "key"))
				assert.False(t, rdb.Exists("key"))

				cache.opt.LocalCache = nil
				assert.NoError(t, cache.Set(item))
				rdb.FastForward(time.Hour * 2)
				want = ""
				assert.ErrorIs(t, cache.Get(context.Background(), "key", &want), ErrCacheMiss)
				assert.Equal(t, "", want)
				assert.False(t, cache.Exists(context.Background(), "key"))
			},
		},
		{
			name: "get and set by skip redis",
			do: func() {
				cache, rdb := initStandaloneCache(t)
				item := &Item{
					Ctx:   context.Background(),
					Key:   "key",
					TTL:   time.Hour,
					Value: "123",
					Skip:  SkipRedis,
				}
				want := ""
				assert.NoError(t, cache.Set(item))
				assert.NoError(t, cache.GetSkip(context.Background(), "key", &want, item.Skip))
				assert.Equal(t, item.Value, want)
				assert.EqualValues(t, cache.Stats().Hits, 0)
				assert.ErrorIs(t, cache.GetSkippingLocalCache(context.Background(), "key", &want), ErrCacheMiss)
				assert.EqualValues(t, cache.Stats().Hits, 0)
				assert.False(t, rdb.Exists("key"))

				cache.opt.Redis = nil
				cache.DeleteFromLocalCache("key")
				//assert.NoError(t, cache.Delete(context.Background(), "key"))
				assert.False(t, cache.Exists(context.Background(), "key"))
			},
		},
		{
			name: "item do",
			do: func() {
				cache, rdb := initStandaloneCache(t)
				item := &Item{
					Ctx: context.Background(),
					Key: "key",
					TTL: time.Hour,
					Do: func(item *Item) (any, error) {
						return "123", nil
					},
				}
				want := ""
				assert.NoError(t, cache.Set(item))
				assert.NoError(t, cache.GetSkip(context.Background(), "key", &want, item.Skip))
				assert.Equal(t, "123", want)
				assert.EqualValues(t, cache.Stats().Hits, 0)
				v, err := rdb.Get("key")
				assert.NoError(t, err)
				assert.Equal(t, "123", v)
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
				cache, rdb := initStandaloneCache(t)
				item := &Item{
					Ctx:   context.Background(),
					Key:   "key",
					TTL:   time.Hour,
					Value: "123",
					Skip:  SkipLocal,
				}
				assert.NoError(t, cache.Set(item))
				err := wctest.RunWait(t, time.Second, func() error {
					want := ""
					item := &Item{
						Ctx:   context.Background(),
						Key:   "key",
						Value: &want,
						Skip:  SkipLocal,
					}
					assert.NoError(t, cache.Once(item))
					assert.Equal(t, "123", want)
					return nil
				}, func() error {
					want := ""
					item := &Item{
						Ctx:   context.Background(),
						Key:   "key",
						Value: &want,
						Skip:  SkipLocal,
					}
					assert.NoError(t, cache.Once(item))
					assert.Equal(t, "123", want)
					return nil
				})
				assert.NoError(t, err)
				assert.EqualValues(t, cache.Stats().Hits, 1)
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
				rdc, rdb := initStandaloneCache(t)
				item := &Item{
					Ctx:   context.Background(),
					Key:   "key",
					TTL:   time.Hour,
					Value: "123",
				}
				assert.NoError(t, rdc.Set(item))
				rdc.CleanLocalCache()
				assert.True(t, rdb.Exists("key"))
				assert.True(t, rdc.Exists(context.Background(), "key"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
		})
	}
}
