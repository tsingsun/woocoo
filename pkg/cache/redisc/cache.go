package redisc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/klauspost/compress/s2"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/singleflight"
)

const (
	compressionThreshold = 64
	timeLen              = 4
)

const (
	noCompression = 0x0
	s2Compression = 0x1
)

const (
	SkipLocal SkipMode = 1 << iota
	SkipRedis

	SkipAll = SkipLocal | SkipRedis
)

var (
	ErrCacheMiss          = errors.New("cache: key is missing")
	errRedisLocalCacheNil = errors.New("cache: both Redis and LocalCache are nil")
)

type SkipMode int

// Any returns true if the skip annotation was set.
func (f SkipMode) Any() bool {
	return f != 0
}

// Is checks if the skip annotation has a specific flag.
func (f SkipMode) Is(mode SkipMode) bool {
	return f&mode != 0
}

type rediser interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd
	SetXX(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.BoolCmd
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.BoolCmd

	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type Item struct {
	Ctx context.Context

	Key   string
	Value interface{}

	// TTL is the cache expiration time.
	// Default TTL is 1 hour.
	TTL time.Duration

	// Do returns value to be cached.
	Do func(*Item) (interface{}, error)

	// SetXX only sets the key if it already exists.
	SetXX bool

	// SetNX only sets the key if it does not already exist.
	SetNX bool

	// SkipFlags indicator skip level.
	Skip SkipMode
}

func (item *Item) Context() context.Context {
	if item.Ctx == nil {
		return context.Background()
	}
	return item.Ctx
}

func (item *Item) value() (interface{}, error) {
	if item.Do != nil {
		return item.Do(item)
	}
	if item.Value != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (item *Item) ttl() time.Duration {
	const defaultTTL = time.Hour

	if item.TTL < 0 {
		return 0
	}

	if item.TTL != 0 {
		if item.TTL < time.Second {
			log.Printf("too short TTL for key=%q: %s", item.Key, item.TTL)
			return defaultTTL
		}
		return item.TTL
	}

	return defaultTTL
}

// ------------------------------------------------------------------------------
type (
	MarshalFunc   func(interface{}) ([]byte, error)
	UnmarshalFunc func([]byte, interface{}) error
)

type Options struct {
	Redis         rediser
	LocalCache    LocalCache
	LocalCacheTTL time.Duration
	StatsEnabled  bool
	Marshal       MarshalFunc
	Unmarshal     UnmarshalFunc
}

type Cache struct {
	opt *Options

	group singleflight.Group

	marshal   MarshalFunc
	unmarshal UnmarshalFunc

	hits   uint64
	misses uint64
}

func NewCache(opt *Options) *Cache {
	cacher := &Cache{
		opt: opt,
	}

	if opt.Marshal == nil {
		cacher.marshal = cacher._marshal
	} else {
		cacher.marshal = opt.Marshal
	}

	if opt.Unmarshal == nil {
		cacher.unmarshal = cacher._unmarshal
	} else {
		cacher.unmarshal = opt.Unmarshal
	}
	return cacher
}

// Set caches the item.
func (cd *Cache) Set(item *Item) error {
	_, _, err := cd.set(item)
	return err
}

func (cd *Cache) set(item *Item) ([]byte, bool, error) {
	value, err := item.value()
	if err != nil {
		return nil, false, err
	}

	b, err := cd.Marshal(value)
	if err != nil {
		return nil, false, err
	}
	if item.Skip == SkipAll {
		return b, true, nil
	}
	if cd.opt.LocalCache != nil && !item.Skip.Is(SkipLocal) {
		cd.opt.LocalCache.Set(item.Key, b)
	}

	if cd.opt.Redis == nil {
		if cd.opt.LocalCache == nil {
			return b, true, errRedisLocalCacheNil
		}
		return b, true, nil
	}

	ttl := item.ttl()
	if ttl == 0 {
		return b, true, nil
	}

	if item.Skip.Is(SkipRedis) {
		return b, true, nil
	}

	if item.SetXX {
		return b, true, cd.opt.Redis.SetXX(item.Context(), item.Key, b, ttl).Err()
	}
	if item.SetNX {
		return b, true, cd.opt.Redis.SetNX(item.Context(), item.Key, b, ttl).Err()
	}
	return b, true, cd.opt.Redis.Set(item.Context(), item.Key, b, ttl).Err()
}

// Exists reports whether value for the given key exists.
func (cd *Cache) Exists(ctx context.Context, key string) bool {
	return cd.Get(ctx, key, nil) == nil
}

// Get gets the value for the given key.
func (cd *Cache) Get(ctx context.Context, key string, value interface{}) error {
	return cd.get(ctx, key, value, SkipMode(0))
}

// GetSkip gets the value for the given key skipping by special.
func (cd *Cache) GetSkip(
	ctx context.Context, key string, value interface{}, mode SkipMode,
) error {
	return cd.get(ctx, key, value, mode)
}

// GetSkippingLocalCache gets the value for the given key skipping local cache.
func (cd *Cache) GetSkippingLocalCache(
	ctx context.Context, key string, value interface{},
) error {
	return cd.get(ctx, key, value, SkipLocal)
}

func (cd *Cache) get(
	ctx context.Context,
	key string,
	value interface{},
	mode SkipMode,
) error {
	b, err := cd.getBytes(ctx, key, mode)
	if err != nil {
		return err
	}
	return cd.unmarshal(b, value)
}

func (cd *Cache) getBytes(ctx context.Context, key string, mode SkipMode) ([]byte, error) {
	if mode == SkipAll {
		return nil, ErrCacheMiss
	}
	if !mode.Is(SkipLocal) && cd.opt.LocalCache != nil {
		b, ok := cd.opt.LocalCache.Get(key)
		if ok {
			return b, nil
		}
	}

	if !mode.Is(SkipRedis) && cd.opt.Redis == nil {
		if cd.opt.LocalCache == nil {
			return nil, errRedisLocalCacheNil
		}
		return nil, ErrCacheMiss
	}

	b, err := cd.opt.Redis.Get(ctx, key).Bytes()
	if err != nil {
		if cd.opt.StatsEnabled {
			atomic.AddUint64(&cd.misses, 1)
		}
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	if cd.opt.StatsEnabled {
		atomic.AddUint64(&cd.hits, 1)
	}

	if !mode.Is(SkipLocal) && cd.opt.LocalCache != nil {
		cd.opt.LocalCache.Set(key, b)
	}
	return b, nil
}

func (cd *Cache) Take(item *Item) error {
	b, cached, err := cd.getSetItemBytes(item, false)
	if err != nil {
		return err
	}

	if item.Value == nil || len(b) == 0 {
		return nil
	}

	if err := cd.unmarshal(b, item.Value); err != nil {
		if cached {
			_ = cd.Delete(item.Context(), item.Key)
			return cd.Take(item)
		}
		return err
	}
	return nil
}

// Once gets the item.Value for the given item.Key from the cache or
// executes, caches, and returns the results of the given item.Func,
// making sure that only one execution is in-flight for a given item.Key
// at a time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
func (cd *Cache) Once(item *Item) error {
	b, cached, err := cd.getSetItemBytes(item, true)
	if err != nil {
		return err
	}

	if item.Value == nil || len(b) == 0 {
		return nil
	}

	if err := cd.unmarshal(b, item.Value); err != nil {
		if cached {
			_ = cd.Delete(item.Context(), item.Key)
			return cd.Once(item)
		}
		return err
	}
	return nil
}

func (cd *Cache) getSetItemBytes(item *Item, once bool) (b []byte, cached bool, err error) {
	if !item.Skip.Is(SkipLocal) && cd.opt.LocalCache != nil {
		b, ok := cd.opt.LocalCache.Get(item.Key)
		if ok {
			return b, true, nil
		}
	}
	var v interface{}
	if once {
		v, err, _ = cd.group.Do(item.Key, func() (interface{}, error) {
			b, cached, err = cd.getSetItem(item)
			return b, err
		})
		if err != nil {
			return nil, false, err
		}
		return v.([]byte), cached, nil
	} else {
		return cd.getSetItem(item)
	}
}

func (cd *Cache) getSetItem(item *Item) (b []byte, cached bool, err error) {
	b, err = cd.getBytes(item.Context(), item.Key, item.Skip)
	if err == nil {
		return b, true, nil
	}

	b, ok, err := cd.set(item)
	if ok {
		return b, false, nil
	}
	return nil, false, err
}

func (cd *Cache) Delete(ctx context.Context, key string) error {
	if cd.opt.LocalCache != nil {
		cd.opt.LocalCache.Del(key)
	}

	if cd.opt.Redis == nil {
		if cd.opt.LocalCache == nil {
			return errRedisLocalCacheNil
		}
		return nil
	}

	_, err := cd.opt.Redis.Del(ctx, key).Result()
	return err
}

func (cd *Cache) DeleteFromLocalCache(key string) {
	if cd.opt.LocalCache != nil {
		cd.opt.LocalCache.Del(key)
	}
}

func (cd *Cache) CleanLocalCache() {
	if cd.opt.LocalCache != nil {
		cd.opt.LocalCache.Clean()
	}
}

func (cd *Cache) Marshal(value interface{}) ([]byte, error) {
	return cd.marshal(value)
}

func (cd *Cache) _marshal(value interface{}) ([]byte, error) {
	switch value := value.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	}

	b, err := msgpack.Marshal(value)
	if err != nil {
		return nil, err
	}

	return compress(b), nil
}

func compress(data []byte) []byte {
	if len(data) < compressionThreshold {
		n := len(data) + 1
		b := make([]byte, n, n+timeLen)
		copy(b, data)
		b[len(b)-1] = noCompression
		return b
	}

	n := s2.MaxEncodedLen(len(data)) + 1
	b := make([]byte, n, n+timeLen)
	b = s2.Encode(b, data)
	b = append(b, s2Compression)
	return b
}

func (cd *Cache) Unmarshal(b []byte, value interface{}) error {
	return cd.unmarshal(b, value)
}

func (cd *Cache) _unmarshal(b []byte, value interface{}) error {
	if len(b) == 0 {
		return nil
	}

	switch value := value.(type) {
	case nil:
		return nil
	case *[]byte:
		clone := make([]byte, len(b))
		copy(clone, b)
		*value = clone
		return nil
	case *string:
		*value = string(b)
		return nil
	}

	switch c := b[len(b)-1]; c {
	case noCompression:
		b = b[:len(b)-1]
	case s2Compression:
		b = b[:len(b)-1]

		var err error
		b, err = s2.Decode(nil, b)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown compression method: %x", c)
	}

	return msgpack.Unmarshal(b, value)
}

//------------------------------------------------------------------------------

type Stats struct {
	Hits   uint64
	Misses uint64
}

// Stats returns cache statistics.
func (cd *Cache) Stats() *Stats {
	if !cd.opt.StatsEnabled {
		return nil
	}
	return &Stats{
		Hits:   atomic.LoadUint64(&cd.hits),
		Misses: atomic.LoadUint64(&cd.misses),
	}
}
