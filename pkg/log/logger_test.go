package log_test

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf, _ = conf.BuildWithOption(conf.LocalPath(testdata.Path("app.yaml")))
	logger = &log.Logger{}
)

func init() {
	logger.Apply(cnf, "log")
}

func TestNewWithOption(t *testing.T) {
	log.Info("get log")
}
