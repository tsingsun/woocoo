package log

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf = conf.New(conf.LocalPath(testdata.TestConfigFile()), conf.BaseDir(testdata.BaseDir())).Load()
)

func TestInfo(t *testing.T) {
	cnf.AsGlobal()
	NewBuiltIn()
	Info("get log")
}
