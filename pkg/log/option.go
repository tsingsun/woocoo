package log

import (
	"github.com/tsingsun/woocoo/pkg/conf"
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
