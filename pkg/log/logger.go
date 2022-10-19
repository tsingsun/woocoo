package log

import (
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	StacktraceKey = "stacktrace"
)

var (
	global      *Logger
	globalApply bool // indicate if you use BuiltIn()
	components  = map[string]*component{}
)

func init() {
	global = New(zap.NewNop())
}

// ContextLogger is functions to help ContextLogger logging,the functions are called each ComponentLogger call the logging method
type ContextLogger interface {
	// LogFields defined how to get logger field from context
	LogFields(logger *Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) []zap.Field
}

// DefaultContextLogger is hold a nothing
type DefaultContextLogger struct {
}

func (n *DefaultContextLogger) LogFields(log *Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) []zap.Field {
	log.Log(lvl, msg, fields...)
	return fields
}

// Logger integrate the Uber Zap library to use in woocoo
//
// if you prefer to golang builtin log style,use log.Info or log.Infof, if zap style,you should use log.Operator().Info()
// if you want to clone Logger,you can call WithOption
type Logger struct {
	*zap.Logger
	WithTraceID bool
	// DisableCaller only indicate the logger weather DisableCaller set by ZapConfigs[0]
	DisableCaller bool `json:"-"`
	// DisableStacktrace only indicate the logger weather DisableCaller, set by ZapConfigs[0]
	DisableStacktrace bool `json:"-"`

	contextLogger ContextLogger
}

// New create an Instance from zap
func New(zl *zap.Logger) *Logger {
	return &Logger{
		Logger:        zl,
		contextLogger: &DefaultContextLogger{},
	}
}

// NewBuiltIn create a logger by configuration,path key is "log", it will be set as global logger
func NewBuiltIn() *Logger {
	if globalApply {
		return global
	}
	pkey := "log"
	if conf.Global().IsSet(pkey) {
		global.Apply(conf.Global().Sub(pkey))
	} else {
		panic("NewBuiltIn:the configuration file does not contain section: log")
	}
	globalApply = true
	return global
}

// Global return the global logger
func Global() *Logger {
	return global
}

// AsGlobal set the Logger as global logger
func (l *Logger) AsGlobal() *Logger {
	globalApply = true
	*global = *l
	zap.ReplaceGlobals(global.Logger)
	return global
}

// Apply implement Configurable interface which can initial from a file used in JSON,YAML
// Logger init trough Apply method will be set as Global.
func (l *Logger) Apply(cfg *conf.Configuration) {
	config, err := NewConfig(cfg)
	if err != nil {
		panic(fmt.Errorf("apply log configuration err:%w", err))
	}
	zl, err := config.BuildZap(zap.AddCallerSkip(config.callerSkip))
	if err != nil {
		panic(fmt.Errorf("apply log configuration err:%w", err))
	}
	l.Logger = zl
	l.WithTraceID = config.WithTraceID
	l.DisableCaller = config.DisableCaller
	l.DisableStacktrace = config.DisableStacktrace
}

// WithOptions clones the current Logger, applies the supplied Options,
// and returns the resulting Logger. It's safe to use concurrently.
func (l *Logger) WithOptions(opts ...zap.Option) *Logger {
	clone := *l
	clone.Logger = l.Logger.WithOptions(opts...)
	return &clone
}

// Operator returns the underlying zap logger.
func (l *Logger) Operator() *zap.Logger {
	return l.Logger
}

// ContextLogger return contextLogger field
func (l *Logger) ContextLogger() ContextLogger {
	return l.contextLogger
}

// SetContextLogger set contextLogger field,if you use the contextLogger,can set or override it.
func (l *Logger) SetContextLogger(f ContextLogger) {
	l.contextLogger = f
}

// Ctx returns a new logger with the context.
func (l *Logger) Ctx(ctx context.Context) *LoggerWithCtx {
	return NewLoggerWithCtx(ctx, l)
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func Sync() error {
	return global.Logger.Sync()
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	global.Logger.Sugar().Debug(args...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	global.Logger.Sugar().Info(args...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	global.Logger.Sugar().Warn(args...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	global.Logger.Sugar().Error(args...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...interface{}) {
	global.Logger.Sugar().DPanic(args...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	global.Logger.Sugar().Panic(args...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	global.Logger.Sugar().Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	global.Logger.Sugar().Debugf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	global.Logger.Sugar().Infof(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	global.Logger.Sugar().Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	global.Logger.Sugar().Errorf(template, args...)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...interface{}) {
	global.Logger.Sugar().DPanicf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	global.Logger.Sugar().Panicf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	global.Logger.Sugar().Fatalf(template, args...)
}
