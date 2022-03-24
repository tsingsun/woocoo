package log

import (
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// ContextHeaderName opentracing log key is trace.traceid
	ContextHeaderName = iota
)

var (
	global      *Logger
	globalApply bool // indicate if use BuiltIn()
	components  = map[string]*component{}
)

func init() {
	global = New(zap.NewNop())
}

// Logger integrate the Uber Zap library to use in woocoo
//
// if you prefer to golang builtin log style,use log.Info or log.Infof, if zap style,you should use log.Operator().Info()
type Logger struct {
	zap *zap.Logger
}

// New create an Instance from zap
func New(zl *zap.Logger) *Logger {
	return &Logger{zap: zl}
}

// NewBuiltIn create a logger by configuration,path key is "log"
func NewBuiltIn() *Logger {
	if globalApply {
		return global
	}
	pkey := "log"
	if conf.Global().IsSet(pkey) {
		global.Apply(conf.Global().Sub(pkey))
	} else {
		StdPrintf("the configuration file does not contain section: log")
	}
	globalApply = true
	return global
}

func Global() *Logger {
	return global
}

func Component(name string) ComponentLogger {
	if cData, ok := components[name]; ok {
		return cData
	}
	c := &component{name}
	components[name] = c
	return c
}

func (l Logger) AsGlobal() {
	global = &l
	zap.ReplaceGlobals(l.zap)
}

func (l Logger) Operator() *zap.Logger {
	return l.zap
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Apply implement Configurable interface which can initial from a file used in JSON,YAML
// Logger init trough Apply method will be set as Global.
func (l *Logger) Apply(cfg *conf.Configuration) {
	config, err := NewConfig(cfg)
	if err != nil {
		panic(fmt.Errorf("apply log configuration err:%w", err))
	}
	zl, err := config.BuildZap()
	if err != nil {
		panic(fmt.Errorf("apply log configuration err:%w", err))
	}

	l.zap = zl
}

func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	l.zap.DPanic(msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zap.Panic(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
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

// Operator return the structured logger
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

func Sync() error {
	return global.zap.Sync()
}
