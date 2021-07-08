package log

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf, _ = conf.BuildWithOption(conf.LocalPath(testdata.Path("app.yaml")))
)

func TestConfig(t *testing.T) {
	c := &Config{}
	if err := c.initConfig(cnf); err != nil {
		t.Error(err)
	}
}
