package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ComponentKey     = "component"
	StacktraceKey    = "stacktrace"
	TraceIDKey       = "trace_id"
	WebComponentName = "web"
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
	builtInFields []zap.Field
}

// Component return a logger with name option base on global logger
func Component(name string, fields ...zap.Field) ComponentLogger {
	if cData, ok := components[name]; ok {
		return cData
	}
	c := &component{name: name, builtInFields: append(fields, zap.String(ComponentKey, name)), l: global}
	components[name] = c
	return c
}

// Logger return component's logger
func (c *component) Logger() *Logger {
	return c.l
}

// SetLogger replace logger, by default component use global logger
func (c *component) SetLogger(logger *Logger) {
	c.l = logger
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
	lc := NewLoggerWithCtx(ctx, c.l)
	lc.fields = c.builtInFields
	return lc
}

// LoggerWithCtx is a wrapper for Logger that also carries a context.Context.
type LoggerWithCtx struct {
	ctx    context.Context
	l      *Logger
	fields []zapcore.Field
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
	if len(c.fields) != 0 {
		fields = append(fields, c.fields...)
	}
	c.l.contextLogger.LogFields(c.l, ctx, lvl, msg, fields)
}

// NewLoggerWithCtx get a logger with context from pool
func NewLoggerWithCtx(ctx context.Context, l *Logger) *LoggerWithCtx {
	return &LoggerWithCtx{ctx: ctx, l: l}
}
