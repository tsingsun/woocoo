package lfu

import (
	"context"
	"fmt"
	"testing"

	"github.com/tsingsun/woocoo/pkg/conf"
)

func newBenchCache() *TinyLFU {
	c, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    "10000",
		"samples": "100000",
		"ttl":     "10m",
	}))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_ = c.Set(ctx, fmt.Sprintf("warm-%d", i), fmt.Sprintf("value-%d", i))
	}
	return c
}

func newBenchCacheWithShards(shards int) *TinyLFU {
	c, err := NewTinyLFU(conf.NewFromStringMap(map[string]any{
		"size":    "10000",
		"samples": "100000",
		"ttl":     "10m",
		"shards":  shards,
	}))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_ = c.Set(ctx, fmt.Sprintf("warm-%d", i), fmt.Sprintf("value-%d", i))
	}
	return c
}

var benchKeys []string

func init() {
	benchKeys = make([]string, 1000)
	for i := range benchKeys {
		benchKeys[i] = fmt.Sprintf("key-%d", i)
	}
}

func BenchmarkTinyLFU_ParallelGet(b *testing.B) {
	c := newBenchCache()
	ctx := context.Background()

	for _, p := range []int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("P%d", p), func(b *testing.B) {
			b.SetParallelism(p)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					var v string
					_ = c.Get(ctx, benchKeys[i%1000], &v)
					i++
				}
			})
		})
	}
}

func BenchmarkTinyLFU_ParallelSet(b *testing.B) {
	c := newBenchCache()
	ctx := context.Background()

	for _, p := range []int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("P%d", p), func(b *testing.B) {
			b.SetParallelism(p)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					_ = c.Set(ctx, benchKeys[i%1000], fmt.Sprintf("val-%d", i))
					i++
				}
			})
		})
	}
}

func BenchmarkTinyLFU_ParallelMixed(b *testing.B) {
	c := newBenchCache()
	ctx := context.Background()

	for _, p := range []int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("P%d", p), func(b *testing.B) {
			b.SetParallelism(p)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					if i%3 == 0 {
						_ = c.Set(ctx, benchKeys[i%1000], fmt.Sprintf("v-%d", i))
					} else {
						var v string
						_ = c.Get(ctx, benchKeys[i%1000], &v)
					}
					i++
				}
			})
		})
	}
}

func BenchmarkTinyLFU_HotKey(b *testing.B) {
	c := newBenchCache()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_ = c.Set(ctx, fmt.Sprintf("hot-%d", i), "value")
	}

	for _, p := range []int{1, 16, 64} {
		b.Run(fmt.Sprintf("P%d", p), func(b *testing.B) {
			b.SetParallelism(p)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					var v string
					_ = c.Get(ctx, fmt.Sprintf("hot-%d", i%100), &v)
					i++
				}
			})
		})
	}
}

// 分片数对比: 固定并发度, 变化分片数
func BenchmarkTinyLFU_ShardCount_Get(b *testing.B) {
	ctx := context.Background()
	for _, shards := range []int{1, 4, 8, 16, 32, 64, 128, 256} {
		c := newBenchCacheWithShards(shards)
		b.Run(fmt.Sprintf("S%d", shards), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					var v string
					_ = c.Get(ctx, benchKeys[i%1000], &v)
					i++
				}
			})
		})
	}
}

func BenchmarkTinyLFU_ShardCount_Set(b *testing.B) {
	ctx := context.Background()
	for _, shards := range []int{1, 4, 8, 16, 32, 64, 128, 256} {
		c := newBenchCacheWithShards(shards)
		b.Run(fmt.Sprintf("S%d", shards), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					_ = c.Set(ctx, benchKeys[i%1000], fmt.Sprintf("val-%d", i))
					i++
				}
			})
		})
	}
}

func BenchmarkTinyLFU_ShardCount_HotKey(b *testing.B) {
	ctx := context.Background()
	for _, shards := range []int{1, 4, 8, 16, 32, 64, 128, 256} {
		c := newBenchCacheWithShards(shards)
		for i := 0; i < 100; i++ {
			_ = c.Set(ctx, fmt.Sprintf("hot-%d", i), "value")
		}
		b.Run(fmt.Sprintf("S%d", shards), func(b *testing.B) {
			b.SetParallelism(64)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					var v string
					_ = c.Get(ctx, fmt.Sprintf("hot-%d", i%100), &v)
					i++
				}
			})
		})
	}
}
