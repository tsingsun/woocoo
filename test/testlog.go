package test

import (
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

// StringWriteSyncer is a WriteSyncer that writes to a string slice for test checking.
type StringWriteSyncer struct {
	Entry []string
}

func (s *StringWriteSyncer) Write(p []byte) (n int, err error) {
	s.Entry = append(s.Entry, string(p))
	return len(p), nil
}

func (s *StringWriteSyncer) Sync() error {
	return nil
}

func (s *StringWriteSyncer) String() string {
	return strings.Join(s.Entry, "/n")
}

type StringCore struct {
	zapcore.Core
}

func NewStringCore(ws *StringWriteSyncer) zapcore.Core {
	en := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	return zapcore.NewCore(en, ws, zap.DebugLevel)
}

func NewStringLogger(ws *StringWriteSyncer, opts ...zap.Option) *zap.Logger {
	// production config contains stacktrace setting
	std, err := zap.NewProduction(opts...)
	if err != nil {
		panic(err)
	}
	core := zapcore.NewTee(std.Core(), NewStringCore(ws))
	return zap.New(core, opts...)
}

func NewGlobalStringLogger() *StringWriteSyncer {
	logdata := &StringWriteSyncer{}
	log.New(NewStringLogger(logdata, zap.AddStacktrace(zap.ErrorLevel))).AsGlobal()
	return logdata
}
