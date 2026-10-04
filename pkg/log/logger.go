package log

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	global          *Logger
	globalComponent ComponentLogger
	components      sync.Map
	globalSugar     *zap.SugaredLogger
)

func init() {
	globalComponent = &component{
		name:      "global",
		useGlobal: true,
	}
	InitGlobalLogger()
}

// ContextLogger is functions to help ContextLogger logging,the functions are called each ComponentLogger call the logging method
type ContextLogger interface {
	// LogFields defined how to log field with context
	LogFields(logger *Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field)
}

// DefaultContextLogger is hold a nothing
type DefaultContextLogger struct {
}

func (n *DefaultContextLogger) LogFields(log *Logger, _ context.Context, lvl zapcore.Level, msg string, fields []zap.Field) {
	log.Log(lvl, msg, fields...)
}

// Logger integrate the Uber Zap library to use in woocoo
//
// if you prefer to golang builtin log style,use log.Info or log.Infof, if zap style,you should use log.Operator().Info()
// if you want to clone Logger,you can call WithOption
// WithTraceID indicate whether to add trace_id field to the log.
//   - web: from X-Request-Id header
//   - grpc: from metadata key "trace_id"
type Logger struct {
	*zap.Logger
	WithTraceID   bool
	TraceIDKey    string
	contextLogger ContextLogger
	// level of zap cores
	logLevels []zap.AtomicLevel
	// sugar caches the SugaredLogger to avoid per-call allocation
	sugar *zap.SugaredLogger
}

// New create an Instance from zap
func New(zl *zap.Logger) *Logger {
	l := &Logger{
		Logger:        zl,
		contextLogger: &DefaultContextLogger{},
		TraceIDKey:    TraceIDKey,
	}
	if zl != nil {
		l.sugar = zl.Sugar()
	}
	return l
}

func InitGlobalLogger() *Logger {
	global = New(zap.Must(zap.NewProduction(zap.AddCallerSkip(CallerSkip))))
	global.AsGlobal()
	return global
}

// NewFromConf create a logger by configuration path "log", it will be set as global logger but run only once.
func NewFromConf(cfg *conf.Configuration) *Logger {
	l := New(nil)
	l.Apply(cfg)
	return l
}

// Global return struct logger if you want to use zap style logging.
func Global() ComponentLogger {
	return globalComponent
}

// AsGlobal set the Logger as global logger
func (l *Logger) AsGlobal() *Logger {
	global = l
	globalSugar = l.Sugar()
	globalComponent.SetLogger(l)
	// reset component,don't reset user defined
	components.Range(func(_, value any) bool {
		cp := value.(*component)
		if cp.useGlobal {
			cp.SetLogger(l)
		}
		return true
	})
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
	l.logLevels = make([]zap.AtomicLevel, len(config.ZapConfigs))
	for i, zc := range config.ZapConfigs {
		l.logLevels[i] = zc.Level
	}
	l.Logger = zl
	l.sugar = zl.Sugar()
	l.WithTraceID = config.WithTraceID
	if config.TraceIDKey != "" {
		l.TraceIDKey = config.TraceIDKey
	}
}

// SetLevel set log level to zap core level.
func (l *Logger) SetLevel(lvl string) error {
	level, err := zapcore.ParseLevel(lvl)
	if err != nil {
		return err
	}
	for _, atomicLevel := range l.logLevels {
		atomicLevel.SetLevel(level)
	}
	return nil
}

// With creates a child logger and adds structured context to it. Fields added
// to the child don't affect the parent, and vice versa.
func (l *Logger) With(fields ...zap.Field) *Logger {
	clone := *l
	clone.Logger = l.Logger.With(fields...)
	clone.sugar = clone.Logger.Sugar()
	return &clone
}

// WithOptions clones the current Logger, applies the supplied Options,
// and returns the resulting Logger. It's safe to use concurrently.
func (l *Logger) WithOptions(opts ...zap.Option) *Logger {
	clone := *l
	clone.Logger = l.Logger.WithOptions(opts...)
	clone.sugar = clone.Logger.Sugar()
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

// Sugar returns the cached SugaredLogger to avoid per-call allocation.
func (l *Logger) Sugar() *zap.SugaredLogger {
	if l.sugar == nil {
		l.sugar = l.Logger.Sugar()
	}
	return l.sugar
}

// Ctx returns a new logger with the context.
func (l *Logger) Ctx(ctx context.Context) *LoggerWithCtx {
	return NewLoggerWithCtx(ctx, l)
}

// IOWriter wrap to Io.Writer which can be used in golang builtin log. Level is the log level which will be written.
func (l *Logger) IOWriter(level zapcore.Level) io.Writer {
	return &Writer{
		Log:   l.Logger,
		Level: level,
	}
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func Sync() error {
	return global.Logger.Sync()
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...any) {
	globalSugar.Debug(args...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...any) {
	globalSugar.Info(args...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...any) {
	globalSugar.Warn(args...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...any) {
	globalSugar.Error(args...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...any) {
	globalSugar.DPanic(args...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...any) {
	globalSugar.Panic(args...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...any) {
	globalSugar.Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...any) {
	globalSugar.Debugf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...any) {
	globalSugar.Infof(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...any) {
	globalSugar.Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...any) {
	globalSugar.Errorf(template, args...)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...any) {
	globalSugar.DPanicf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...any) {
	globalSugar.Panicf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...any) {
	globalSugar.Fatalf(template, args...)
}
