package lfu

import (
	"context"
	"errors"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/vmihailenco/go-tinylfu"
	"math/rand"
	"reflect"
	"sync"
	"time"
)

const (
	maxOffset      = 10 * time.Second
	defaultSamples = 100000
)

var (
	ErrValueReceiverNil = errors.New("cache: value receiver must not nil pointer")
)

var _ cache.Cache = (*TinyLFU)(nil)

// TinyLFU is a cache implementation of TinyLFU algorithm. It forces the cache data to have an expiration time.
//
// Default ttl is 1 minute.Notice that the ttl will be less the setting,
// randomly reduced by a value between 0 and the offset.
type TinyLFU struct {
	mu            sync.Mutex
	rand          *rand.Rand
	lfu           *tinylfu.T
	ttl           time.Duration
	offset        time.Duration
	deviation     int64
	size, samples int

	marshal   cache.MarshalFunc
	unmarshal cache.UnmarshalFunc
}

// Apply implements the conf.Configurable interface
func (c *TinyLFU) Apply(cnf *conf.Configuration) error {
	c.size = cnf.Int("size")
	c.samples = cnf.Int("samples")
	c.ttl = cnf.Duration("ttl")
	if c.ttl == 0 {
		c.ttl = time.Minute
	}
	c.offset = c.ttl / time.Duration(c.deviation)
	if c.offset > maxOffset {
		c.offset = maxOffset
	}
	if c.samples == 0 {
		c.samples = defaultSamples
	}
	c.lfu = tinylfu.New(c.size, c.samples)
	return nil
}

func NewTinyLFU(cnf *conf.Configuration) (*TinyLFU, error) {
	lfu := TinyLFU{
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec
		deviation: 10,
	}
	if err := lfu.Apply(cnf); err != nil {
		return nil, err
	}
	if lfu.marshal == nil {
		lfu.marshal = cache.DefaultMarshalFunc
		lfu.unmarshal = cache.DefaultUnmarshalFunc
	}
	return &lfu, nil
}

// Get returns the value for the given key, or ErrCacheMiss. If the value is nil, the value will not be set
func (c *TinyLFU) Get(ctx context.Context, key string, value any, opts ...cache.Option) (err error) {
	opt := cache.ApplyOptions(opts...)
	return c.GetInner(ctx, key, value, opt)
}

func (c *TinyLFU) GetInner(ctx context.Context, key string, value any, opt *cache.Options) error {
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
	if !opt.Raw {
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
	return c.SetInner(ctx, key, value, opt.TTL, opt.Raw)
}

// SetInner sets the value for the given key.ttl is the expiration time, if ttl is zero, the default ttl will be used.
func (c *TinyLFU) SetInner(ctx context.Context, key string, value any, ttl time.Duration, raw bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.ttl
	}
	if ttl >= c.ttl && c.offset > 0 {
		ttl -= time.Duration(c.rand.Int63n(int64(c.offset)))
	} else {
		ttl -= time.Duration(c.rand.Int63n(int64(ttl) / c.deviation))
	}
	if raw {
		c.lfu.Set(&tinylfu.Item{Key: key, Value: value, ExpireAt: time.Now().Add(ttl)})
		return nil
	}
	v, err := c.marshal(value)
	if err != nil {
		return err
	}
	c.lfu.Set(&tinylfu.Item{Key: key, Value: v, ExpireAt: time.Now().Add(ttl)})
	return nil
}

func (c *TinyLFU) Has(ctx context.Context, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.lfu.Get(key)
	return ok
}

func (c *TinyLFU) Del(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lfu.Del(key)
	return nil
}

func (c *TinyLFU) IsNotFound(err error) bool {
	return errors.Is(err, cache.ErrCacheMiss)
}

func (c *TinyLFU) Clean() {
	c.lfu = tinylfu.New(c.size, c.samples)
}
