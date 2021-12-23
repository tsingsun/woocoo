package log_test

import (
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf    = testdata.Config
	logger = &log.Logger{}
)

func TestInfo(t *testing.T) {
	testdata.Config.AsGlobal()
	log.NewBuiltIn()
	log.Info("get log")
}
