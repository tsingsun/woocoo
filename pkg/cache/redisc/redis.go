package redisc

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/cache/lfu"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/store/redisx"
	"golang.org/x/sync/singleflight"
)

var _ cache.Cache = (*Redisc)(nil)

type (
	// Redisc is a cache implementation of redis.
	//
	// if you want to register to cache manager, set a `driverName` in configuration
	Redisc struct {
		driverName string
		redis      redis.Cmdable
		local      *lfu.TinyLFU
		stats      *cache.Stats
		// marshal and unmarshal func called when cross process cache
		marshal   cache.MarshalFunc
		unmarshal cache.UnmarshalFunc

		group singleflight.Group
	}

	Option func(*Redisc)
)

func WithRedisClient(cli redis.UniversalClient) Option {
	return func(redisc *Redisc) {
		redisc.redis = cli
	}
}

// New creates a new redis cache with the provided configuration.
//
// Cache Configuration:
//
//	driverName: redis # optional, default is redis
//	addrs: # required
//	db: 0
//	... # other redis configuration
//	local: # local cache,optional, default is nil
//	  size: 1000 # optional, default is 1000
//	  samples: 100000 # optional, default is 100000
//	  ttl: 1m # optional, default is 1m
//
// If you want to register to cache manager, set a `driverName` in configuration.
func New(cfg *conf.Configuration, opts ...Option) (*Redisc, error) {
	c := &Redisc{
		driverName: "redis",
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.Apply(cfg); err != nil {
		return nil, err
	}
	return c, nil
}

// Register cache to cache manager
func (cd *Redisc) Register() error {
	return cache.RegisterCache(cd.driverName, cd)
}

// Apply conf.configurable
func (cd *Redisc) Apply(cfg *conf.Configuration) (err error) {
	if cfg.Bool("stats") {
		cd.stats = &cache.Stats{}
	}
	if cfg.IsSet("local") {
		cd.local, err = lfu.NewTinyLFU(cfg.Sub("local"))
		if err != nil {
			return err
		}
	}
	if cd.redis == nil {
		if cfg.IsSet("addrs") {
			remote, err := redisx.NewClient(cfg)
			if err != nil {
				return err
			}
			cd.redis = remote
		}
	}
	if k := cfg.String("driverName"); k != "" {
		cd.driverName = k
		if err = cd.Register(); err != nil {
			return err
		}
	}
	if cd.redis == nil && cd.local == nil {
		err = errors.New("redis cache must have a redis client or local cache")
	}
	if cd.marshal == nil {
		cd.marshal = cache.DefaultMarshalFunc
	}
	if cd.unmarshal == nil {
		cd.unmarshal = cache.DefaultUnmarshalFunc
	}
	return
}

// Get returns the value associated with the given key.
func (cd *Redisc) Get(ctx context.Context, key string, v any, opts ...cache.Option) error {
	opt := cache.ApplyOptions(opts...)
	if opt.Group || opt.Getter != nil {
		return cd.Group(ctx, key, v, opt)
	}
	_, err := cd.get(ctx, key, v, opt)
	return err
}

// Group is a method use singleflight to get value
func (cd *Redisc) Group(ctx context.Context, key string, v any, opt *cache.Options) error {
	marshal, cached, err := cd.getSetItemGroup(ctx, key, v, opt)
	if err != nil {
		return err
	}
	if marshal != nil {
		if err = cd.unmarshal(marshal, v); err != nil {
			if cached {
				cd.local.Del(ctx, key) //nolint:errcheck
			}
			return err
		}
	}
	return nil
}

func (cd *Redisc) getRemoteData(ctx context.Context, key string, mode cache.SkipMode) (data []byte, err error) {
	if mode.Is(cache.SkipRemote) {
		return nil, cache.ErrCacheMiss
	}

	data, err = cd.redis.Get(ctx, key).Bytes()
	if err != nil {
		cd.stats.AddMiss()
		if errors.Is(err, redis.Nil) {
			return nil, cache.ErrCacheMiss
		}
		return nil, err
	}
	cd.stats.AddHit()
	return data, nil
}

// get gets the value for the given key. if loaded the value has marshaled.
func (cd *Redisc) get(ctx context.Context, key string, value any, opt *cache.Options) (cached bool, err error) {
	mode := opt.Skip
	local := cd.local != nil && !mode.Is(cache.SkipLocal)
	if local {
		err := cd.local.Get(ctx, key, value)
		if err == nil {
			return true, nil
		}
	}

	b, err := cd.getRemoteData(ctx, key, mode)
	if err != nil {
		return false, err
	}
	if err = cd.unmarshal(b, value); err != nil {
		return false, err
	}
	if local {
		cd.local.SetInner(ctx, key, value, opt.TTL, opt.Raw) //nolint:errcheck
	}
	return true, nil
}

func (cd *Redisc) getSetItemGroup(ctx context.Context, key string, value any, opt *cache.Options) (marshal []byte, cached bool, err error) {
	// first try to load from local cache
	if !opt.Skip.Is(cache.SkipLocal) && cd.local != nil {
		err := cd.local.GetInner(ctx, key, value, opt)
		if err == nil {
			return nil, true, nil
		}
	}

	v, err, _ := cd.group.Do(key, func() (any, error) {
		data, err := cd.getRemoteData(ctx, key, opt.Skip)
		if errors.Is(err, cache.ErrCacheMiss) {
			if opt.Getter == nil {
				return nil, err
			}
			gv, err := opt.Getter(ctx, key)
			if err != nil {
				return nil, err
			}
			data, cached, err = cd.set(ctx, key, gv, opt)
			if err != nil {
				return nil, err
			}
			return data, nil
		}

		return data, err
	})
	if err != nil {
		return nil, false, err
	}
	return v.([]byte), cached, nil
}

// Set sets the value associated with the given key.
// if ttl < 0 ,will not save to redis,but save to local cache if enabled
func (cd *Redisc) Set(ctx context.Context, key string, v any, opts ...cache.Option) error {
	opt := cache.ApplyOptions(opts...)
	_, _, err := cd.set(ctx, key, v, opt)
	return err
}

// Set sets the value associated with the given key.
// if ttl < 0 ,will not save to redis,but save to local cache if enabled
func (cd *Redisc) set(ctx context.Context, key string, v any, opt *cache.Options) (marshaled []byte, cached bool, err error) {
	ttl := opt.Expiration()
	if !opt.Skip.Is(cache.SkipRemote) {
		if marshaled, err = cd.marshal(v); err != nil {
			return
		}
		var ok bool
		switch {
		case opt.SetXX:
			ok, err = cd.redis.SetXX(ctx, key, marshaled, ttl).Result()
			if !ok && err == nil {
				err = fmt.Errorf("setxx: key not exist:%s", key)
			}
		case opt.SetNX:
			ok, err = cd.redis.SetNX(ctx, key, marshaled, ttl).Result()
			if !ok && err == nil {
				err = fmt.Errorf("setnx key already exist:%s", key)
			}
		default:
			err = cd.redis.Set(ctx, key, marshaled, ttl).Err()
		}
	} else if !opt.Raw {
		if marshaled, err = cd.marshal(v); err != nil {
			return
		}
	}

	local := cd.local != nil && !opt.Skip.Is(cache.SkipLocal)
	if local && err == nil {
		if opt.Raw {
			cd.local.SetInner(ctx, key, v, ttl, true) //nolint:errcheck
		} else {
			cd.local.SetInner(ctx, key, marshaled, ttl, true) //nolint:errcheck
		}
		cached = true
	}
	return
}

// Has returns true if the given key exists.
func (cd *Redisc) Has(ctx context.Context, key string) bool {
	if cd.local != nil && cd.local.Has(ctx, key) {
		return true
	}

	return cd.redis.Exists(ctx, key).Val() != 0
}

// Del deletes the given key.
func (cd *Redisc) Del(ctx context.Context, key string) error {
	cd.DeleteFromLocalCache(key)
	_, err := cd.redis.Del(ctx, key).Result()
	return err
}

// IsNotFound returns true if the error is cache.ErrCacheMiss.
func (cd *Redisc) IsNotFound(err error) bool {
	return errors.Is(err, cache.ErrCacheMiss)
}

// RedisClient returns the underlying redis client.
func (cd *Redisc) RedisClient() redis.Cmdable {
	return cd.redis
}

// LocalCacheEnabled returns true if local cache is enabled.
func (cd *Redisc) LocalCacheEnabled() bool {
	return cd.local != nil
}

func (cd *Redisc) CleanLocalCache() {
	if cd.local != nil {
		cd.local.Clean()
	}
}

func (cd *Redisc) DeleteFromLocalCache(key string) {
	if cd.local != nil {
		cd.local.Del(context.Background(), key) //nolint:errcheck
	}
}

func (cd *Redisc) Stats() *cache.Stats {
	return cd.stats
}
