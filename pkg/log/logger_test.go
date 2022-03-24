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

func TestLogger_With(t *testing.T) {
	type fields struct {
		logger ComponentLogger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "global", fields: fields{logger: global.zap}},
		{name: "component-1", fields: fields{logger: Component("component-1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.fields.logger
			l.Info("test")
		})
	}
}
