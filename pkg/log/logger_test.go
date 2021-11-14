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

func TestNewWithOption(t *testing.T) {
	log.Info("get log")
}
