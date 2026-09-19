package lfu

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
)

func TestNewTinyLFU(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		cnfstr := `
driverName: local
size: 1000
samples: 10000000
ttl: 10m
deviation: 10
subsidiary: true
`
		cnf := conf.NewFromBytes([]byte(cnfstr))
		c, err := NewTinyLFU(cnf)
		require.NoError(t, err)
		assert.Equal(t, 1000, c.Size)
		assert.Equal(t, 10000000, c.Samples)
		assert.Equal(t, 10*time.Minute, c.TTL)
		assert.Equal(t, int64(10), c.Deviation)
		assert.Equal(t, true, c.Subsidiary)
		assert.Equal(t, 10*time.Second, c.offset)
		assert.Equal(t, "local", c.DriverName)
		lc, _ := cache.GetCache("local")
		assert.NotNil(t, lc)
		_, err = NewTinyLFU(cnf)
		assert.Error(t, err, "repeat register")
	})
	t.Run("ttl format", func(t *testing.T) {
		cnfstr := `
ttl: "string"
`
		cnf := conf.NewFromBytes([]byte(cnfstr))
		_, err := NewTinyLFU(cnf)
		assert.ErrorContains(t, err, `'ttl' time: invalid duration`)
	})
}

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
	mycache.Wait()

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
		ctx := context.Background()
		assert.NoError(t, local.Set(ctx, "key", "value"))
		local.Wait()
		var v string
		err := local.Get(ctx, "key", &v)
		assert.NoError(t, err)
		assert.Equal(t, "value", v)
		err = local.Get(ctx, "key", &v)
		assert.NoError(t, err, "repeat get to check if del for expired")

		assert.NoError(t, local.Set(ctx, "key", "value", cache.WithTTL(time.Second)))
		local.Wait()
		assert.True(t, local.Has(ctx, "key"))
		time.Sleep(time.Second)
		err = local.Get(ctx, "key", &v)
		assert.ErrorIs(t, err, cache.ErrCacheMiss)

		var count int
		geterFunc := func(ctx context.Context, key string) (any, error) {
			count++
			return "getter" + strconv.Itoa(count), nil
		}
		err = local.Get(ctx, "getterKey", &v, cache.WithGetter(geterFunc), cache.WithTTL(time.Second))
		assert.NoError(t, err)
		assert.Equal(t, "getter1", v)
		local.Wait()
		err = local.Get(ctx, "getterKey", &v)
		assert.NoError(t, err)
		assert.Equal(t, "getter1", v)

		err = local.Get(ctx, "pointer", v, cache.WithGetter(geterFunc), cache.WithTTL(time.Second))
		assert.ErrorIs(t, err, cache.ErrReceiverMustPointer)
	})

	t.Run("get native value", func(t *testing.T) {
		ctx := context.Background()
		var s string
		assert.NoError(t, local.Set(ctx, "key", "value"))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &s))
		assert.Equal(t, "value", s)
		assert.NoError(t, local.Set(ctx, "key", "valueRaw", cache.WithRaw()))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &s, cache.WithRaw()))
		assert.Equal(t, "valueRaw", s)
		assert.NoError(t, local.Get(ctx, "getterS", &s, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return "getterS", nil
			})))
		assert.Equal(t, "getterS", s)
		local.Wait()

		var v []byte
		assert.NoError(t, local.Set(ctx, "key", []byte("value")))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &v))
		assert.Equal(t, []byte("value"), v)
		assert.NoError(t, local.Set(ctx, "key", []byte("rawValue"), cache.WithRaw()))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &v, cache.WithRaw()))
		assert.Equal(t, []byte("rawValue"), v)
		assert.NoError(t, local.Get(ctx, "getterBS", &v, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return []byte("getter"), nil
			})))
		assert.Equal(t, []byte("getter"), v)
		local.Wait()

		var m map[string]string
		assert.NoError(t, local.Set(ctx, "key", map[string]string{"name": "value"}, cache.WithRaw()))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &m, cache.WithRaw()))
		assert.Equal(t, m, map[string]string{"name": "value"})
		assert.NoError(t, local.Get(ctx, "getterM", &m, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return map[string]string{"name": "value"}, nil
			})))
		assert.Equal(t, map[string]string{"name": "value"}, m)
		local.Wait()

		var i int
		assert.NoError(t, local.Set(ctx, "key", 1, cache.WithRaw()))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &i, cache.WithRaw()))
		assert.Equal(t, 1, i)
		assert.NoError(t, local.Get(ctx, "getterI", &i, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return 2, nil
			})))
		assert.Equal(t, 2, i)
		local.Wait()

		var f float64
		assert.NoError(t, local.Set(ctx, "key", 1.1, cache.WithRaw()))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &f, cache.WithRaw()))
		assert.Equal(t, 1.1, f)
		assert.NoError(t, local.Get(ctx, "getterF", &f, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return 2.0, nil
			})))
		assert.Equal(t, 2.0, f)
		local.Wait()

		var b bool
		assert.NoError(t, local.Set(ctx, "key", true, cache.WithRaw()))
		local.Wait()
		require.NoError(t, local.Get(ctx, "key", &b, cache.WithRaw()))
		assert.Equal(t, true, b)
		assert.NoError(t, local.Get(ctx, "getterB", &b, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return true, nil
			})))
		assert.Equal(t, true, b)
		local.Wait()
	})
	t.Run("nil value", func(t *testing.T) {
		err := local.Get(context.Background(), "key", nil)
		assert.ErrorIs(t, err, ErrValueReceiverNil)
	})
	t.Run("del", func(t *testing.T) {
		assert.NoError(t, local.Set(context.Background(), "key", []byte("value")))
		local.Wait()
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
		ctx := context.Background()
		var (
			tt  T
			ttp *T
		)
		assert.NoError(t, local.Get(ctx, "key", &tt, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return T{Name: "getter"}, nil
			})))
		assert.Equal(t, T{Name: "getter"}, tt)
		local.Wait()

		assert.NoError(t, local.Get(ctx, "keypoint", &ttp, cache.WithGetter(
			func(ctx context.Context, key string) (any, error) {
				return &T{Name: "pointer"}, nil
			})))
		assert.Equal(t, "pointer", ttp.Name)
		local.Wait()

		require.NoError(t, local.Set(ctx, "key", &T{Name: "pointer"}, cache.WithRaw()))
		local.Wait()
		err := local.Get(ctx, "key", T{Name: "pointer"}, cache.WithRaw())
		assert.ErrorIs(t, err, cache.ErrReceiverMustPointer)

		var v *T
		err = local.Get(ctx, "key", &v, cache.WithRaw())
		assert.NoError(t, err)
		assert.Equal(t, "pointer", v.Name)
		v.Name = "value2"
		var v1 *T
		err = local.Get(ctx, "key", &v1, cache.WithRaw())
		assert.NoError(t, err)
		assert.Equal(t, "value2", v1.Name, "point same value")

		require.NoError(t, local.Set(ctx, "key", T{Name: "struct"}))
		local.Wait()
		var v2 T
		require.NoError(t, local.Get(ctx, "key", &v2))
		assert.Equal(t, "struct", v2.Name)
		v2.Name = "value3"
		var v3 T
		require.NoError(t, local.Get(ctx, "key", &v3))
		assert.Equal(t, "struct", v3.Name, "value type must not change")
	})
	t.Run("subsidiary", func(t *testing.T) {
		subs, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
			"size":       "100000",
			"samples":    "100000",
			"ttl":        "2s",
			"subsidiary": true,
		}))
		require.NoError(t, err)
		require.NoError(t, subs.Set(context.Background(), "ttl", "123", cache.WithTTL(time.Second*-1)))
		subs.Wait()
		time.Sleep(time.Second*2 + time.Millisecond*500)
		require.ErrorIs(t, subs.Get(context.Background(), "ttl", ""), cache.ErrCacheMiss)

		want := ""
		require.NoError(t, subs.Set(context.Background(), "key", "123", cache.WithTTL(time.Hour)))
		subs.Wait()
		time.Sleep(time.Second * 3)
		assert.ErrorIs(t, subs.Get(context.Background(), "key", &want), cache.ErrCacheMiss)
		require.NoError(t, subs.Set(context.Background(), "key", "123", cache.WithTTL(time.Second)))
		subs.Wait()
		time.Sleep(time.Second*1 + time.Millisecond*200)
		assert.ErrorIs(t, subs.Get(context.Background(), "key", &want), cache.ErrCacheMiss)
	})
}

func TestTinyLFU_Set(t *testing.T) {
	t.Run("setInner", func(t *testing.T) {
		local, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
			"size":    "100000",
			"samples": "100000",
			"ttl":     "1s",
		}))
		require.NoError(t, err)
		ctx := context.Background()
		assert.NoError(t, local.SetInner(ctx, "key", "value", time.Second*2,
			&cache.Options{Raw: false}))
		assert.NoError(t, local.SetInner(ctx, "key1", "value", time.Second*3,
			&cache.Options{Raw: true, Skip: cache.SkipRemote}))
		local.Wait()
		time.Sleep(time.Second * 2)
		want := ""
		assert.Error(t, local.Get(ctx, "key", &want))
		assert.NoError(t, local.Get(ctx, "key1", &want, cache.WithRaw()))
	})
	t.Run("setNX", func(t *testing.T) {
		local, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
			"size":    "100000",
			"samples": "100000",
		}))
		require.NoError(t, err)
		// NX Only set the key if it does not already exist.
		ctx := context.Background()
		assert.NoError(t, local.Set(ctx, "key", "value"))
		local.Wait()
		assert.Error(t, local.Set(ctx, "key", "value", cache.WithSetNX()))
		assert.NoError(t, local.Set(ctx, "key1", "value", cache.WithSetNX()))
		local.Has(ctx, "key")
		local.Has(ctx, "key1")
	})
	t.Run("setXX", func(t *testing.T) {
		local, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
			"size":    "100000",
			"samples": "100000",
		}))
		require.NoError(t, err)
		// XX Only set the key if it already exists.
		ctx := context.Background()
		assert.Error(t, local.Set(ctx, "key", "value", cache.WithSetXX()))
		assert.False(t, local.Has(ctx, "key"))
		assert.NoError(t, local.Set(ctx, "key", "value"))
		local.Wait()
		assert.NoError(t, local.Set(ctx, "key", "value", cache.WithSetXX()))
	})
}

// 压力测试: 大量写入不应导致内存爆炸.
func TestTinyLFU_OOM(t *testing.T) {
	keys := make([]string, 10000)
	for i := range keys {
		keys[i] = randWord()
	}

	c, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    "1000",
		"samples": "10000",
	}))
	require.NoError(t, err)

	for i := 0; i < 5e6; i++ {
		key := keys[i%len(keys)]
		c.SetInner(context.Background(), key, key, 0, &cache.Options{Raw: true})
	}
}

// 并发读写 + TTL 过期场景下数据完整性.
func TestTinyLFU_ConcurrentReadWriteWithTTL(t *testing.T) {
	const size = 50000

	strFor := func(i int) string {
		return fmt.Sprintf("a string %d", i)
	}
	keyName := func(i int) string {
		return fmt.Sprintf("key-%00000d", i)
	}

	mycache, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    "1000",
		"samples": "10000",
	}))
	require.NoError(t, err)

	for i := 0; i < size; i++ {
		key := keyName(i)
		mycache.SetInner(context.Background(), key, []byte(strFor(i)), time.Second, &cache.Options{Raw: true})
	}
	mycache.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := ctx.Done()
loop:
	for {
		select {
		case <-done:
			break loop
		default:
			i := rand.Intn(size)
			key := keyName(i)

			var b []byte
			err := mycache.GetInner(context.Background(), key, &b, true)
			if err != nil {
				continue loop
			}

			got := string(b)
			expected := strFor(i)
			if got != expected {
				t.Fatalf("expected=%q got=%q key=%q", expected, got, key)
			}
		}
	}
}

const cacheTestTotalCapacity = 1000

// 对同 key 先后写入(等待处理完成), 应获取到最新值.
// 注意: ristretto 的 Set 是异步的, 快速连续 Set 同 key 不保证 last-write-wins.
// 此测试验证: 每次 Set 后等待处理完成, 再 Set 新值, 值正确更新.
func TestTinyLFU_UpdateSameKey(t *testing.T) {
	localCache, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    fmt.Sprintf("%d", cacheTestTotalCapacity),
		"samples": fmt.Sprintf("%d", cacheTestTotalCapacity),
	}))
	require.NoError(t, err)
	ctx := context.Background()

	// 设置初始值
	assert.NoError(t, localCache.Set(ctx, "key", "v1", cache.WithRaw()))
	localCache.Wait()
	var got string
	require.NoError(t, localCache.GetInner(ctx, "key", &got, true))
	assert.Equal(t, "v1", got)

	// 更新值
	assert.NoError(t, localCache.Set(ctx, "key", "v2", cache.WithRaw()))
	localCache.Wait()
	require.NoError(t, localCache.GetInner(ctx, "key", &got, true))
	assert.Equal(t, "v2", got)

	// 再次更新
	assert.NoError(t, localCache.Set(ctx, "key", "v3", cache.WithRaw()))
	localCache.Wait()
	require.NoError(t, localCache.GetInner(ctx, "key", &got, true))
	assert.Equal(t, "v3", got)

	// 其他 key 不受影响
	assert.NoError(t, localCache.Set(ctx, "other", "other-val", cache.WithRaw()))
	localCache.Wait()
	var other string
	require.NoError(t, localCache.GetInner(ctx, "other", &other, true))
	assert.Equal(t, "other-val", other)
	require.NoError(t, localCache.GetInner(ctx, "key", &got, true))
	assert.Equal(t, "v3", got)
}

// SetNX 对同 key 的同步语义: 第一次成功, 后续拒绝.
// 这验证了 signer 等场景所需的防重放语义.
func TestTinyLFU_SetNXDuplicateKey(t *testing.T) {
	localCache, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    fmt.Sprintf("%d", cacheTestTotalCapacity),
		"samples": fmt.Sprintf("%d", cacheTestTotalCapacity),
	}))
	require.NoError(t, err)
	ctx := context.Background()

	// 第一次 SetNX 应成功
	assert.NoError(t, localCache.Set(ctx, "sig", nil, cache.WithSetNX(), cache.WithTTL(time.Hour)))
	// 第二次 SetNX 同 key 应失败 (同步语义, 无需等待)
	assert.Error(t, localCache.Set(ctx, "sig", nil, cache.WithSetNX(), cache.WithTTL(time.Hour)))

	// 其他 key 的 SetNX 不受影响
	assert.NoError(t, localCache.Set(ctx, "other-sig", nil, cache.WithSetNX(), cache.WithTTL(time.Hour)))

	// Has 应返回 true
	assert.True(t, localCache.Has(ctx, "sig"))
	assert.True(t, localCache.Has(ctx, "other-sig"))
}

// 旧的同 key 数据过期后, 新写入应可正常读取.
func TestTinyLFU_ExpiredKeyThenNewWrite(t *testing.T) {
	localCache, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    fmt.Sprintf("%d", cacheTestTotalCapacity),
		"samples": fmt.Sprintf("%d", cacheTestTotalCapacity),
	}))
	require.NoError(t, err)
	ctx := context.Background()

	// 设置短 TTL 的值
	assert.NoError(t, localCache.Set(ctx, "key", "old", cache.WithRaw(), cache.WithTTL(100*time.Millisecond)))
	localCache.Wait()
	var got string
	require.NoError(t, localCache.GetInner(ctx, "key", &got, true))
	assert.Equal(t, "old", got)

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 过期后 Get 应返回 miss
	err = localCache.GetInner(ctx, "key", &got, true)
	assert.ErrorIs(t, err, cache.ErrCacheMiss)

	// 写入新值
	assert.NoError(t, localCache.Set(ctx, "key", "new", cache.WithRaw(), cache.WithTTL(time.Hour)))
	localCache.Wait()
	require.NoError(t, localCache.GetInner(ctx, "key", &got, true))
	assert.Equal(t, "new", got)

	// 其他 key 不受影响
	assert.NoError(t, localCache.Set(ctx, "other", "other-val", cache.WithRaw(), cache.WithTTL(time.Hour)))
	localCache.Wait()
	var other string
	require.NoError(t, localCache.GetInner(ctx, "other", &other, true))
	assert.Equal(t, "other-val", other)
}

func randWord() string {
	buf := make([]byte, 64)
	io.ReadFull(cryptorand.Reader, buf)
	return string(buf)
}
