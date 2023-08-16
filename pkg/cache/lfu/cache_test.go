package lfu

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"math/rand"
	"testing"
	"time"
)

func TestTinyLFU_Get_CorruptionOnExpiry(t *testing.T) {
	strFor := func(i int) string {
		return fmt.Sprintf("a string %d", i)
	}
	keyName := func(i int) string {
		return fmt.Sprintf("key-%00000d", i)
	}

	mycache, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    "100000",
		"samples": "100000",
	}))
	require.NoError(t, err)
	size := 50000
	// Put a bunch of stuff in the cache with a TTL of 1 second
	for i := 0; i < size; i++ {
		key := keyName(i)
		mycache.Set(context.Background(), key, strFor(i))
	}

	// Read stuff for a bit longer than the TTL - that's when the corruption occurs
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := ctx.Done()
loop:
	for {
		select {
		case <-done:
			// this is expected
			break loop
		default:
			i := rand.Intn(size)
			key := keyName(i)
			s := ""
			err := mycache.Get(context.Background(), key, &s)
			if mycache.IsNotFound(err) {
				continue loop
			}
			assert.Equal(t, strFor(i), s)
		}
	}
	mycache.Clean()
}

func TestTinyLFU_Get(t *testing.T) {
	local, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    "100000",
		"samples": "100000",
	}))
	require.NoError(t, err)
	t.Parallel()
	t.Run("string with ttl", func(t *testing.T) {
		assert.NoError(t, local.Set(context.Background(), "key", "value"))
		var v string
		err := local.Get(context.Background(), "key", &v)
		assert.NoError(t, err)
		assert.Equal(t, "value", v)

		assert.NoError(t, local.Set(context.Background(), "key", "value", cache.WithTTL(time.Second)))
		assert.True(t, local.Has(context.Background(), "key"))
		time.Sleep(time.Second)
		err = local.Get(context.Background(), "key", &v)
		assert.ErrorIs(t, err, cache.ErrCacheMiss)
	})

	t.Run("get native value", func(t *testing.T) {
		ctx := context.Background()
		var s string
		assert.NoError(t, local.Set(ctx, "key", "value"))
		require.NoError(t, local.Get(ctx, "key", &s))
		assert.Equal(t, "value", s)

		var v []byte
		assert.NoError(t, local.Set(ctx, "key", []byte("value")))
		require.NoError(t, local.Get(ctx, "key", &v))
		assert.Equal(t, []byte("value"), v)

		var m map[string]string
		assert.NoError(t, local.Set(ctx, "key", map[string]string{"name": "value"}, cache.WithRaw()))
		require.NoError(t, local.Get(ctx, "key", &m, cache.WithRaw()))
		assert.Equal(t, m, map[string]string{"name": "value"})

		var i int
		assert.NoError(t, local.Set(ctx, "key", 1, cache.WithRaw()))
		require.NoError(t, local.Get(ctx, "key", &i, cache.WithRaw()))
		assert.Equal(t, 1, i)

		var f float64
		assert.NoError(t, local.Set(ctx, "key", 1.1, cache.WithRaw()))
		require.NoError(t, local.Get(ctx, "key", &f, cache.WithRaw()))
		assert.Equal(t, 1.1, f)

		var b bool
		assert.NoError(t, local.Set(ctx, "key", true, cache.WithRaw()))
		require.NoError(t, local.Get(ctx, "key", &b, cache.WithRaw()))
		assert.Equal(t, true, b)
	})
	t.Run("nil value", func(t *testing.T) {
		err := local.Get(context.Background(), "key", nil)
		assert.ErrorIs(t, err, ErrValueReceiverNil)
	})
	t.Run("del", func(t *testing.T) {
		assert.NoError(t, local.Set(context.Background(), "key", []byte("value")))
		var v []byte
		err := local.Get(context.Background(), "key", &v)
		assert.NoError(t, err)
		assert.Equal(t, []byte("value"), v)
		assert.NoError(t, local.Del(context.Background(), "key"))
	})
	t.Run("pointer and struct changes", func(t *testing.T) {
		type T struct {
			Name string
		}
		require.NoError(t, local.Set(context.Background(), "key", &T{Name: "pointer"}, cache.WithRaw()))
		err := local.Get(context.Background(), "key", T{Name: "pointer"}, cache.WithRaw())
		require.ErrorContains(t, err, "output value must be a pointer")

		var v *T
		err = local.Get(context.Background(), "key", &v, cache.WithRaw())
		assert.NoError(t, err)
		assert.Equal(t, "pointer", v.Name)
		v.Name = "value2"
		var v1 *T
		err = local.Get(context.Background(), "key", &v1, cache.WithRaw())
		assert.NoError(t, err)
		assert.Equal(t, "value2", v1.Name, "point same value")

		require.NoError(t, local.Set(context.Background(), "key", T{Name: "struct"}))
		var v2 T
		require.NoError(t, local.Get(context.Background(), "key", &v2))
		assert.Equal(t, "struct", v2.Name)
		v2.Name = "value3"
		var v3 T
		require.NoError(t, local.Get(context.Background(), "key", &v3))
		assert.Equal(t, "struct", v3.Name, "value type must not change")
	})
}
