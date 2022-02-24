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
	TraceIdKey        = "traceid"
	SpanIdKey         = "spanid"
)

//Logger integrate the Uber Zap library to use in woocoo
type Logger struct {
	zap *zap.Logger
}

var global, _ = New(zap.NewNop())

func New(zl *zap.Logger) (*Logger, error) {
	return &Logger{zap: zl}, nil
}

// NewBuiltIn create a logger by configuration
func NewBuiltIn() *Logger {
	l := &Logger{}
	l.Apply(conf.Global().Sub("log"), "")
	l.AsGlobal()
	return l
}

func Global() *Logger {
	return global
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
func (l *Logger) Apply(cfg *conf.Configuration, path string) {
	config, err := NewConfig(cfg)
	if err != nil {
		panic(fmt.Errorf("%s apply from configuration file err:%s", "log", err))
	}
	zl, err := config.BuildZap()
	if err != nil {
		panic(fmt.Errorf("%s apply from configuration file err:%s", "log", err))
	}

	l.zap = zl
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

// get the structured logger
func Operator() *zap.Logger {
	return global.zap
}

func Debug(args ...interface{}) {
	global.zap.Sugar().Debug(args...)
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

func Info(args ...interface{}) {
	global.zap.Sugar().Info(args...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

func Warn(args ...interface{}) {
	global.zap.Sugar().Warn(args...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

func Error(args ...interface{}) {
	global.zap.Sugar().Error(args...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

func DPanic(args ...interface{}) {
	global.zap.Sugar().DPanic(args...)
}

func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	l.zap.DPanic(msg, fields...)
}

func Panic(args ...interface{}) {
	global.zap.Sugar().Panic(args...)
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zap.Panic(msg, fields...)
}

func Fatal(args ...interface{}) {
	global.zap.Sugar().Fatal(args...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
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
