package cache

import (
	"context"
	"time"
)

type Option func(*Options)

type Options struct {
	// TTL is the cache expiration time.
	TTL time.Duration
	// Getter returns value to be cached.call getter to get a value(such as query database) and set cache using c.expiry
	Getter func(ctx context.Context, key string) (any, error)
	// SetXX only sets the key if it already exists.
	SetXX bool
	// SetNX only sets the key if it does not already exist.
	SetNX bool
	// SkipFlags indicator skip level.
	Skip SkipMode
	// Raw indicates whether to skip serialization. default is false to keep coroutine safe.
	// Caches accessed across processes are serialized, that flag Generally used for memory cache.
	//
	// false that means serialize value to Item.V. if true, Item.V is raw value but support by implemented Cache.
	// has implemented Cache: lfu cache
	Raw bool
	// Group indicates whether to singleflight.
	Group bool
}

func ApplyOptions(opts ...Option) *Options {
	opt := &Options{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

func (o *Options) Expiration() time.Duration {
	if o.TTL == 0 || o.TTL >= time.Second {
		return o.TTL
	}
	return defaultItemTTL
}

// WithTTL sets the cache expiration time.
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// WithGetter sets the cache getter.
func WithGetter(getter func(ctx context.Context, key string) (any, error)) Option {
	return func(o *Options) {
		o.Getter = getter
	}
}

// WithSetXX sets the cache SetXX.
func WithSetXX() Option {
	return func(o *Options) {
		o.SetXX = true
	}
}

// WithSetNX sets the cache SetNX.
func WithSetNX() Option {
	return func(o *Options) {
		o.SetNX = true
	}
}

// WithSkip sets the cache Skip.
func WithSkip(skip SkipMode) Option {
	return func(o *Options) {
		o.Skip = skip
	}
}

// WithRaw sets the cache Raw.
func WithRaw() Option {
	return func(o *Options) {
		o.Raw = true
	}
}

// WithGroup sets the cache Group.
func WithGroup() Option {
	return func(o *Options) {
		o.Group = true
	}
}
