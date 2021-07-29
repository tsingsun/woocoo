package log

import (
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf = testdata.Config
)

func TestConfig(t *testing.T) {
	c := &Config{}
	if err := c.initConfig(cnf); err != nil {
		t.Error(err)
	}
}
