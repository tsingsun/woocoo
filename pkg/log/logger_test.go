package log

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"testing"
)

var (
	cnf = conf.New(conf.WithLocalPath(testdata.TestConfigFile()), conf.WithBaseDir(testdata.BaseDir())).Load()
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

func TestLogger_TextEncode(t *testing.T) {
	var cfgStr = `
development: true
log:
  disableTimestamp: false
  disableErrorVerbose: false
  cores:
    - level: debug
      disableCaller: false
      disableStacktrace: false
      encoding: text
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).Load()
	got, err := NewConfig(cfg.Sub("log"))
	assert.NoError(t, err)
	zl, err := got.BuildZap()
	assert.NoError(t, err)
	logger := New(zl)
	logger.Info("info")
	//TOOD: github action bug for output error
	//logger.Error("error msg", zap.Error(fmt.Errorf("error")))
	logger.Info("info for scalar", zap.String("string", "it's a string"), zap.Int("int", 1), zap.Int64("int64", 1), zap.Duration("duration", 1))
	logger.Info("info for object", zap.Any("object", testdata.TestStruct()))
}
