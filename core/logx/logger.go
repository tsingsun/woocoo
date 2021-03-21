package logx

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"github.com/tsingsun/woocoo/core/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"path/filepath"
	"strings"
)

const (
	// opentracing log key is trace.traceid
	ContextHeaderName = iota
	TraceIdKey        = "traceid"
	SpanIdKey         = "spanid"
)

//Logger integrate the Uber Zap library to use in woocoo
type Logger struct {
	opts       options
	zap        *zap.Logger
	ToZapField func(values []interface{}) []zapcore.Field
}

var global *Logger

func init() {
	core := zapcore.NewTee(zapcore.NewNopCore())
	zapLogger := zap.New(core)
	global = &Logger{zap: zapLogger}
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Configurable implement Configurable interface which can initial from a file used in JSON,YAML
func (l *Logger) Configurable(config *conf.Config, configKey string) {
	if &l.opts == nil {
		l.opts = defaultOptions
	}
	l.opts.config = config
	var optFunc []Option
	if configKey != "" {
		le, ok := l.opts.config.Get(configKey).([]interface{})
		if !ok {
			panic("configuration file error at log.")
		}
		for _, v := range le {
			if vv, ok := v.(map[interface{}]interface{}); ok {
				switch strings.ToLower(vv["type"].(string)) {
				case "file":
					ll := lumberjack.Logger{}
					if err := mapstructure.Decode(vv, &ll); err != nil {
						panic(err)
					}
					if ll.Filename == "" {
						panic("log configuration miss filename")
					}
					if f, err := filepath.Abs(ll.Filename); err != nil {
						ll.Filename = f
					}
					optFunc = append(optFunc, LogRotate(&ll, vv["level"].(int)))
				case "std":
					optFunc = append(optFunc, Std(vv["level"].(int)))
				}
			}
		}
	}
	for _, o := range optFunc {
		o(&l.opts)
	}
	l.initCore()
}

func (l *Logger) initCore() {
	if len(l.opts.cores) == 0 {
		l.opts.cores = append(l.opts.cores, zapcore.NewNopCore())
	}
	core := zapcore.NewTee(l.opts.cores...)
	zapLogger := zap.New(core)
	l.zap = zapLogger
}

func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

// NewContext creates a new context the given contextual fields
func NewContext(ctx context.Context, fields ...zapcore.Field) context.Context {
	return context.WithValue(ctx, ContextHeaderName, WithContext(ctx).With(fields...))
}

// WithContext returns a logger from the given context
func WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return global
	}
	if ctxLogger, ok := ctx.Value(ContextHeaderName).(*Logger); ok {
		return ctxLogger
	}
	return global
}

// get trace id of zap field type
func TraceIdField(ctx context.Context) zap.Field {
	val, _ := ctx.Value(TraceIdKey).(string)
	return zap.String(TraceIdKey, val)
}

// build global logger with option
func NewWithOption(opt ...Option) *Logger {
	global.opts = defaultOptions
	for _, o := range opt {
		o(&global.opts)
	}
	global.initCore()
	return global
}

// get the structured logger
func Operator() *zap.Logger {
	return global.zap
}

func Debug(args ...interface{}) {
	global.zap.Sugar().Debug(args...)
}

func Info(args ...interface{}) {
	global.zap.Sugar().Info(args...)
}

func Warn(args ...interface{}) {
	global.zap.Sugar().Warn(args...)
}

func Error(args ...interface{}) {
	global.zap.Sugar().Error(args...)
}

func DPanic(args ...interface{}) {
	global.zap.Sugar().DPanic(args...)
}

func Panic(args ...interface{}) {
	global.zap.Sugar().Panic(args...)
}

func Fatal(args ...interface{}) {
	global.zap.Sugar().Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	global.zap.Sugar().Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	global.zap.Sugar().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	global.zap.Sugar().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	global.zap.Sugar().Errorf(template, args...)
}

func DPanicf(template string, args ...interface{}) {
	global.zap.Sugar().DPanicf(template, args...)
}

func Panicf(template string, args ...interface{}) {
	global.zap.Sugar().Panicf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	global.zap.Sugar().Fatalf(template, args...)
}
