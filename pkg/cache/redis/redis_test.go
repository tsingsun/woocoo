package redis_test

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/tsingsun/woocoo/pkg/cache/redis"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
	"time"
)

var (
	cnf = testdata.Config
	mr  *miniredis.Miniredis
)

func initCache(t *testing.T, remote bool) *redis.Cache {
	var err error
	cache := &redis.Cache{}
	if !remote {
		if mr, err = miniredis.Run(); err != nil {
			t.Error(err)
		}
		cnf.Parser().Set("cache.redis.addr", mr.Addr())
	}
	//single node
	cache.Apply(cnf, "cache")
	return cache
}

func TestCache_Apply(t *testing.T) {
	cache := &redis.Cache{}
	cache.Apply(cnf, "cache")
	if cache == nil {
		t.Error("apply cache error")
	}
}

func TestCache_Take(t *testing.T) {
	defer mr.Close()
	cache := initCache(t, false)
	got := "string"
	want := "abc"
	err := cache.Take(&got, "string", time.Minute, func() (interface{}, error) {
		return want, nil
	})
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("got %v,but want %v", got, want)
	}
}

func TestCache_Set(t *testing.T) {
	cache := initCache(t, true)
	got := "string"
	if err := cache.Set("string", got, time.Hour); err != nil {
		t.Error(err)
	}
	var want string
	if err := cache.Get("string", &want); err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("got %v,but want %v", got, want)
	}

}
