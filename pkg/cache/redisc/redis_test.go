package redisc_test

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/cache/redisc"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
	"time"
)

var (
	cfg = conf.New(conf.WithLocalPath(testdata.TestConfigFile()), conf.WithBaseDir(testdata.BaseDir())).Load()
)

func initStandaloneCache(t *testing.T) (*redisc.Redisc, *miniredis.Miniredis) {
	cfgstr := `
cache:
  redis:
    driverName: 
    #    type: cluster
    #    addrs:
    #      - 127.0.0.1:6379
    type: standalone
    addr: 127.0.0.1:6379
    db: 1
    local:
      size: 1000
      ttl: 60
`
	cfg := conf.NewFromBytes([]byte(cfgstr)).Load()
	mr := miniredis.RunT(t)
	cfg.Parser().Set("cache.redis.addr", mr.Addr())
	cfg.Parser().Set("cache.redis.driverName", mr.Addr())
	cache := redisc.NewBuiltIn()
	return cache, mr
}

func TestCache_Apply(t *testing.T) {
	b := []byte(`
local:
  redis:
    local:
      size: 1000
      ttl: 60s
standalone:
  redis:
    type: standalone
    addr: 127.0.0.1:6379
    db: 1
    local:
      size: 1000
      ttl: 60s
cluster:
  redis:
    type: cluster
    addrs:
    - 127.0.0.1:6379  
    db: 1
    local:
      size: 1000
      ttl: 60s
`)
	err := cfg.ParserFromBytes(b)
	if err != nil {
		panic(err)
	}
	for _, s := range []string{"local", "standalone", "cluster"} {
		t.Run(s, func(t *testing.T) {
			cache := &redisc.Redisc{}
			cache.Apply(cfg.Sub(s + ".redis"))
		})
	}
}

func TestCache_Take(t *testing.T) {
	cache, mr := initStandaloneCache(t)
	defer mr.Close()
	got := "string"
	want := "abc"
	err := cache.Take(&got, "string", time.Minute, func() (interface{}, error) {
		return want, nil
	})
	assert.NoError(t, err)
	if got != want {
		t.Errorf("got %v,but want %v", got, want)
	}
}

func TestCache_Set(t *testing.T) {
	cache, mr := initStandaloneCache(t)
	defer mr.Close()
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
