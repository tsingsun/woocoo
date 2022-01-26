package redis

import (
	"github.com/go-redis/redis/v8"
	"github.com/tsingsun/woocoo/pkg/conf"
	"testing"
)

func TestNewConfig(t *testing.T) {
	b := `
store:
  redis1:
    type: cluster
    addrs:
      - 127.0.0.1:6379
  redis2:
    type: standalone
    addr: 127.0.0.1:6379
    db: 1
`
	cfg := conf.NewFromBytes([]byte(b)).Load()
	t.Run("cluster", func(t *testing.T) {
		r1 := &Client{}
		r1.Apply(cfg, "store.redis1")
	})
	t.Run("standalone", func(t *testing.T) {
		r1 := &Client{}
		r1.Apply(cfg, "store.redis2")

		if o, ok := r1.option.(*redis.Options); !ok {
			t.Errorf("option mismatch,want:redis.option,but got:%v", o)
		} else {
			if o.DB != 1 {
				t.Error("db field error")
			}
		}
	})
}
