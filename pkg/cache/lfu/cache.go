package lfu

import (
	"context"
	"errors"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/vmihailenco/go-tinylfu"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

const (
	defaultTTL     = time.Minute
	maxOffset      = 10 * time.Second
	defaultSamples = 100000
)

var (
	ErrValueReceiverNil = errors.New("cache: value receiver must not nil pointer")
)

var _ cache.Cache = (*TinyLFU)(nil)

// Config is the configuration for TinyLFU cache
type Config struct {
	// DriverName set it to register to cache manager.
	DriverName string `yaml:"driverName" json:"driverName"`
	Size       int    `yaml:"size" json:"size"`
	Samples    int    `yaml:"samples" json:"samples"`
	// TTL is default to set item ttl, if you use no expired cache, this value is not used.
	TTL       time.Duration `yaml:"ttl" json:"ttl"`
	Deviation int64         `yaml:"deviation" json:"deviation"`
	// Subsidiary indicate whether the cache is a subsidiary cache,
	// if true, the cache will not be registered to cache manager and ttl will be the max ttl.
	Subsidiary bool `yaml:"subsidiary" json:"subsidiary"`
}

// TinyLFU is a cache implementation of TinyLFU algorithm. It forces the cache data to have an expiration time.
//
// Default ttl is 1 minute.Notice that the ttl will be less the setting,
// randomly reduced by a value between 0 and the offset.
type TinyLFU struct {
	Config
	mu     sync.Mutex
	rand   *rand.Rand
	lfu    *tinylfu.T
	offset time.Duration

	marshal   cache.MarshalFunc
	unmarshal cache.UnmarshalFunc
}

// Register cache to cache manager
func (c *TinyLFU) Register() error {
	return cache.RegisterCache(c.DriverName, c)
}

// Apply implements the conf.Configurable interface
func (c *TinyLFU) Apply(cnf *conf.Configuration) error {
	if err := cnf.Unmarshal(&c.Config); err != nil {
		return err
	}
	if c.Subsidiary {
		c.offset = c.TTL / time.Duration(c.Deviation)
		if c.offset > maxOffset {
			c.offset = maxOffset
		}
	}
	if c.DriverName != "" {
		if err := c.Register(); err != nil {
			return err
		}
	}
	c.lfu = tinylfu.New(c.Size, c.Samples)
	return nil
}

func NewTinyLFU(cnf *conf.Configuration) (*TinyLFU, error) {
	c := TinyLFU{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec
		Config: Config{
			Samples:   defaultSamples,
			Deviation: 10,
			TTL:       defaultTTL,
		},
	}
	if err := c.Apply(cnf); err != nil {
		return nil, err
	}

	if c.marshal == nil {
		c.marshal = cache.DefaultMarshalFunc
		c.unmarshal = cache.DefaultUnmarshalFunc
	}
	return &c, nil
}

// Get returns the value for the given key, or ErrCacheMiss. If the value is nil, the value will not be set
func (c *TinyLFU) Get(ctx context.Context, key string, value any, opts ...cache.Option) (err error) {
	opt := cache.ApplyOptions(opts...)
	return c.GetInner(ctx, key, value, opt.Raw)
}

func (c *TinyLFU) GetInner(_ context.Context, key string, value any, raw bool) error {
	if value == nil {
		return ErrValueReceiverNil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	val, ok := c.lfu.Get(key)
	if !ok {
		return cache.ErrCacheMiss
	}
	if val == nil {
		return nil
	}
	if !raw {
		v, ok := val.([]byte)
		if !ok {
			return errors.New("cache: can't unmarshal,value must be []byte")
		}
		return c.unmarshal(v, value)
	}
	switch value := value.(type) {
	case *string:
		*value = val.(string)
	case *[]byte:
		*value = val.([]byte)
	case *bool:
		*value = val.(bool)
	case *int:
		*value = val.(int)
	case *float64:
		*value = val.(float64)
	default:
		if reflect.TypeOf(value).Kind() != reflect.Ptr {
			return errors.New("cache: output value must be a pointer")
		}
		reflect.ValueOf(value).Elem().Set(reflect.ValueOf(val))
	}
	return nil
}

// Set sets the value for the given key.ttl is the expiration time, if ttl is zero, the default ttl will be used.
// the ttl will be less the setting,randomly reduced by a value between 0 and the offset.
func (c *TinyLFU) Set(ctx context.Context, key string, value any, opts ...cache.Option) error {
	opt := cache.ApplyOptions(opts...)
	return c.setOptions(ctx, key, value, opt.TTL, opt)
}

func (c *TinyLFU) setOptions(_ context.Context, key string, value any, ttl time.Duration, opt *cache.Options) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ttl = c.fixTTL(ttl, opt)

	switch {
	case opt.SetXX:
		if _, ok := c.lfu.Get(key); !ok {
			return fmt.Errorf("setxx: key not exist:%s", key)
		}
	case opt.SetNX:
		if _, ok := c.lfu.Get(key); ok {
			return fmt.Errorf("setnx key already exist:%s", key)
		}
	}
	return c.setValue(key, value, ttl, opt.Raw)
}

// skip remote cache is mean that only set local cache as not a subsidiary cache temporarily,
// that ttl can greater than default c.TTL
func (c *TinyLFU) fixTTL(ttl time.Duration, opt *cache.Options) time.Duration {
	if c.Subsidiary && !opt.Skip.Is(cache.SkipRemote) {
		if ttl > c.TTL {
			ttl = c.TTL
		}
	}
	if ttl <= 0 {
		ttl = c.TTL
	}
	if c.offset > 0 {
		if ttl >= c.TTL {
			ttl += time.Duration(c.rand.Int63n(int64(c.offset)))
		} else {
			ttl += time.Duration(c.rand.Int63n(int64(ttl) / c.Deviation))
		}
	}
	return ttl
}

func (c *TinyLFU) setValue(key string, value any, ttl time.Duration, raw bool) error {
	exp := time.Time{}
	if ttl != 0 {
		exp = time.Now().Add(ttl)
	}
	if raw {
		c.lfu.Set(&tinylfu.Item{Key: key, Value: value, ExpireAt: exp})
		return nil
	}
	v, err := c.marshal(value)
	if err != nil {
		return err
	}
	c.lfu.Set(&tinylfu.Item{Key: key, Value: v, ExpireAt: exp})
	return nil
}

// SetInner sets the value for the given key.ttl is the expiration time, if ttl is zero, the default ttl will be used.
func (c *TinyLFU) SetInner(_ context.Context, key string, value any, ttl time.Duration, opt *cache.Options) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ttl = c.fixTTL(ttl, opt)
	return c.setValue(key, value, ttl, opt.Raw)
}

func (c *TinyLFU) Has(_ context.Context, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.lfu.Get(key)
	return ok
}

func (c *TinyLFU) Del(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lfu.Del(key)
	return nil
}

func (c *TinyLFU) IsNotFound(err error) bool {
	return errors.Is(err, cache.ErrCacheMiss)
}

func (c *TinyLFU) Clean() {
	c.lfu = tinylfu.New(c.Size, c.Samples)
}
