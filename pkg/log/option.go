package log

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Option func(*options)

type options struct {
	config *conf.Config
	cores  []zapcore.Core
}

var defaultOptions = options{
	cores: []zapcore.Core{},
}

func getLevel(cLevel int) zapcore.Level {
	level := zap.InfoLevel
	if cLevel >= -1 && cLevel <= 5 {
		level = levelConv(int8(cLevel))
	}
	return level
}

func levelConv(level int8) zapcore.Level {
	switch level {
	case -1:
		return zap.DebugLevel
	case 0:
		return zap.InfoLevel
	case 1:
		return zap.WarnLevel
	case 2:
		return zap.ErrorLevel
	case 3:
		return zap.DPanicLevel
	case 4:
		return zap.PanicLevel
	case 5:
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}
