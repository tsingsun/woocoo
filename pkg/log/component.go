package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
)

// ComponentLogger is sample and base using for component that also carries a context.Context. It uses the global logger.
type ComponentLogger interface {
	Logger() *Logger
	SetLogger(logger *Logger)
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	DPanic(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Ctx(ctx context.Context) *LoggerWithCtx
}

type component struct {
	name          string
	l             *Logger
	cl            *Logger
	builtInFields []zap.Field
	// useGlobal is true if the component is using the global logger.
	useGlobal bool
}

// Component return a logger with name option base on a logger.
//
// The logger will be lazy set up,Using the global logger by default.
func Component(name string, fields ...zap.Field) ComponentLogger {
	if cData, ok := components[name]; ok {
		return cData
	}
	c := &component{
		name:          name,
		builtInFields: append(fields, zap.String(ComponentKey, name)),
		useGlobal:     true,
	}
	components[name] = c
	return c
}

// Logger return component's logger
func (c *component) Logger() *Logger {
	if c.l == nil {
		c.SetLogger(global)
	}
	return c.l
}

// SetLogger replace logger, by default component use global logger
func (c *component) SetLogger(logger *Logger) {
	c.useGlobal = logger == global
	if c.l != logger {
		c.l = logger
		c.cl = c.l.WithOptions(zap.AddCallerSkip(CallerSkip + 1))
	}
}

func (c *component) Debug(msg string, fields ...zap.Field) {
	c.l.Debug(msg, append(fields, c.builtInFields...)...)
}

func (c *component) Info(msg string, fields ...zap.Field) {
	c.l.Info(msg, append(fields, c.builtInFields...)...)
}

func (c *component) Warn(msg string, fields ...zap.Field) {
	c.l.Warn(msg, append(fields, c.builtInFields...)...)
}

func (c *component) Error(msg string, fields ...zap.Field) {
	c.l.Error(msg, append(fields, c.builtInFields...)...)
}

func (c *component) DPanic(msg string, fields ...zap.Field) {
	c.l.DPanic(msg, append(fields, c.builtInFields...)...)
}

func (c *component) Panic(msg string, fields ...zap.Field) {
	c.l.Panic(msg, append(fields, c.builtInFields...)...)
}

func (c *component) Fatal(msg string, fields ...zap.Field) {
	c.l.Fatal(msg, append(fields, c.builtInFields...)...)
}

// Ctx returns a new logger with the context.
func (c *component) Ctx(ctx context.Context) *LoggerWithCtx {
	lc := NewLoggerWithCtx(ctx, c.cl)
	lc.fields = c.builtInFields
	return lc
}

// loggerWithCtxPool
var (
	loggerWithCtxPool = sync.Pool{
		New: func() any {
			return &LoggerWithCtx{}
		},
	}
	GetLoggerWithCtx = func(ctx context.Context, l *Logger) *LoggerWithCtx {
		lc := loggerWithCtxPool.Get().(*LoggerWithCtx)
		lc.ctx = ctx
		lc.l = l
		return lc
	}
	PutLoggerWithCtx = func(lc *LoggerWithCtx) {
		lc.ctx = nil
		lc.l = nil
		loggerWithCtxPool.Put(lc)
	}
)

// LoggerWithCtx is a wrapper for Logger that also carries a context.Context.
type LoggerWithCtx struct {
	ctx    context.Context
	l      *Logger
	fields []zapcore.Field
}

// WithOptions reset the logger with options.
func (c *LoggerWithCtx) WithOptions(opts ...zap.Option) *LoggerWithCtx {
	c.l = c.l.WithOptions(opts...)
	return c
}

func (c *LoggerWithCtx) Debug(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.DebugLevel, msg, fields)
}

func (c *LoggerWithCtx) Info(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.InfoLevel, msg, fields)
}

func (c *LoggerWithCtx) Warn(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.WarnLevel, msg, fields)
}

func (c *LoggerWithCtx) Error(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.ErrorLevel, msg, fields)
}

func (c *LoggerWithCtx) DPanic(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.DPanicLevel, msg, fields)
}

func (c *LoggerWithCtx) Panic(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.PanicLevel, msg, fields)
}

func (c *LoggerWithCtx) Fatal(msg string, fields ...zap.Field) {
	c.logFields(c.ctx, zap.FatalLevel, msg, fields)
}

func (c *LoggerWithCtx) Log(lvl zapcore.Level, msg string, fields []zap.Field) {
	c.logFields(c.ctx, lvl, msg, fields)
}

func (c *LoggerWithCtx) logFields(ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) {
	defer PutLoggerWithCtx(c)
	if len(c.fields) != 0 {
		fields = append(fields, c.fields...)
	}
	c.l.contextLogger.LogFields(c.l, ctx, lvl, msg, fields)
}

// NewLoggerWithCtx get a logger with context from pool
func NewLoggerWithCtx(ctx context.Context, l *Logger) *LoggerWithCtx {
	return GetLoggerWithCtx(ctx, l)
}
