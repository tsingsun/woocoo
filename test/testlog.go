package test

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"sync"
)

// StringWriteSyncer is a WriteSyncer that writes to a string slice for test checking.
type StringWriteSyncer struct {
	Entry []string
	mu    sync.RWMutex
}

func (s *StringWriteSyncer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Entry = append(s.Entry, string(p))
	return len(p), nil
}

func (s *StringWriteSyncer) Sync() error {
	return nil
}

func (s *StringWriteSyncer) String() string {
	return strings.Join(s.Entry, "/n")
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
