package testlog

import (
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strconv"
)

func ApplyGlobal(disableStacktrace bool) {
	glog := log.InitGlobalLogger()
	glog.Apply(conf.NewFromBytes([]byte(fmt.Sprintf(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: %s`, strconv.FormatBool(disableStacktrace)))))
	glog.AsGlobal()
}

func InitStringWriteSyncer(opts ...zap.Option) *test.StringWriteSyncer {
	opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	logdata := &test.StringWriteSyncer{}
	zl := test.NewStringLogger(logdata, opts...)
	glog := log.Global().Logger()
	opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zl.Core()
	}))
	glog.WithOptions(opts...).AsGlobal()
	return logdata
}
