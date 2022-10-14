package test

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

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

func NewStringLogger(ws *StringWriteSyncer) *zap.Logger {
	std := zap.NewExample()
	core := zapcore.NewTee(std.Core(), NewStringCore(ws))
	return zap.New(core)
}
