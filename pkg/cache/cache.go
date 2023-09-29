package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/klauspost/compress/s2"
	"github.com/vmihailenco/msgpack/v5"
	"sync/atomic"
	"time"
)

const (
	CacheKindRedis = "redis"

	defaultItemTTL = time.Hour

	timeLen              = 4
	compressionThreshold = 64
	noCompression        = 0x0
	s2Compression        = 0x1
)

var (
	ErrDriverNameMiss = errors.New("cache: driverName is empty")
	ErrCacheMiss      = errors.New("cache: key is missing")
)

// Cache is the interface for cache.
type Cache interface {
	// Get gets the value from cache and unmarshal it to v. Make sure the value is a pointer and zero.
	Get(ctx context.Context, key string, value any, opts ...Option) error
	// Set sets the value to cache.
	Set(ctx context.Context, key string, value any, opts ...Option) error
	// Has reports whether the value for the given key exists.
	Has(ctx context.Context, key string) bool
	// Del deletes the value for the given key.
	Del(ctx context.Context, key string) error
	// IsNotFound detect the error weather not found from cache
	IsNotFound(err error) bool
}

// SkipMode controls the cache load which level from a combined cache .
type SkipMode int

const (
	SkipLocal SkipMode = 1 << iota
	SkipRemote
	// SkipCache skip cache load,means load from source
	SkipCache = SkipLocal | SkipRemote
)

// Any returns true if the skip annotation was set.
func (f SkipMode) Any() bool {
	return f != 0
}

// Is checks if the skip annotation has a specific flag.
func (f SkipMode) Is(mode SkipMode) bool {
	return f&mode != 0
}

// Stats is the redis cache analyzer.
type Stats struct {
	Hits   uint64
	Misses uint64
}

func (s *Stats) AddHit() {
	if s == nil {
		return
	}
	atomic.AddUint64(&s.Hits, 1)
}

func (s *Stats) AddMiss() {
	if s == nil {
		return
	}
	atomic.AddUint64(&s.Misses, 1)
}

type (
	MarshalFunc   func(any) ([]byte, error)
	UnmarshalFunc func([]byte, any) error
)

func DefaultMarshalFunc(value any) ([]byte, error) {
	switch value := value.(type) {
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	case nil:
		return nil, nil
	}

	b, err := msgpack.Marshal(value)
	if err != nil {
		return nil, err
	}

	return compress(b), nil
}

func DefaultUnmarshalFunc(b []byte, value any) error {
	if len(b) == 0 {
		return nil
	}

	switch value := value.(type) {
	case *[]byte:
		clone := make([]byte, len(b))
		copy(clone, b)
		*value = clone
		return nil
	case *string:
		*value = string(b)
		return nil
	case nil:
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
