package redis_test

import (
	"github.com/go-redis/redis/v8"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf, _ = conf.BuildWithOption(conf.LocalPath(testdata.Path("app.yaml")))
)

func TestNewConfig(t *testing.T) {
	cc, err := cnf.Parser().Sub("cache")
	if err != nil {
		t.Error(err)
	}
	config := redis.ClusterOptions{}
	err = cc.UnmarshalByJson("redis", &config)
	if err != nil {
		t.Error(err)
	}
	t.Log(config)
}
