package redisc

import (
	"math/rand"
	"sync"
	"time"

	"github.com/vmihailenco/go-tinylfu"
)

// LocalCache is a local cache to store a marshalled data in memory.
type LocalCache interface {
	// Set sets a value for a key. ttl maybe has a rule in Local Cache
	Set(key string, data []byte, ttl time.Duration)
	Get(key string) ([]byte, bool)
	Del(key string)
	// Clean removes all items from the cache in memory.
	Clean()
}

type TinyLFU struct {
	mu     sync.Mutex
	rand   *rand.Rand
	lfu    *tinylfu.T
	ttl    time.Duration
	offset time.Duration

	size, samples int
}

var _ LocalCache = (*TinyLFU)(nil)

func NewTinyLFU(size int, ttl time.Duration) *TinyLFU {
	const maxOffset = 10 * time.Second

	offset := ttl / 10
	if offset > maxOffset {
		offset = maxOffset
	}

	c := &TinyLFU{
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
		ttl:     ttl,
		offset:  offset,
		size:    size,
		samples: 100000,
	}
	c.lfu = tinylfu.New(c.size, c.samples)
	return c
}

func (c *TinyLFU) UseRandomizedTTL(offset time.Duration) {
	c.offset = offset
}

func (c *TinyLFU) Set(key string, b []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.ttl
		if c.offset > 0 {
			ttl += time.Duration(c.rand.Int63n(int64(c.offset)))
		}
	}

	c.lfu.Set(&tinylfu.Item{
		Key:      key,
		Value:    b,
		ExpireAt: time.Now().Add(ttl),
	})
}

func (c *TinyLFU) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, ok := c.lfu.Get(key)
	if !ok {
		return nil, false
	}

	b := val.([]byte)
	return b, true
}

func (c *TinyLFU) Del(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lfu.Del(key)
}

func (c *TinyLFU) Clean() {
	c.lfu = tinylfu.New(c.size, c.samples)
}
