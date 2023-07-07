package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
)

// ComponentLogger is sample and base using for component that also carries a context.Context. It uses the global logger.
type (
	ComponentLogger interface {
		// Logger return component's logger.
		// Notice GetComponentLoggerOption
		//   if you want to get the original logger, you can use WithOriginalLogger() option.
		//   If you want to get the logger with context, you can use WithContextLogger() option.
		//   otherwise, you will get the logger with build in fields.
		Logger(opts ...GetComponentLoggerOption) *Logger
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

	GetComponentLoggerOption func(*GetComponentLoggerOptions)

	GetComponentLoggerOptions struct {
		originalLogger bool // if true,return logger without build in fields
		ctxLogger      bool // if true,return logger with context
	}

	component struct {
		name          string
		logger        *Logger
		l             *Logger
		cl            *Logger
		builtInFields []zap.Field
		// useGlobal is true if the component is using the global logger.
		useGlobal bool
	}
)

// WithOriginalLogger returns the original logger in the ComponentLogger.
func WithOriginalLogger() GetComponentLoggerOption {
	return func(options *GetComponentLoggerOptions) {
		options.originalLogger = true
	}
}

// WithContextLogger returns the logger with context in the ComponentLogger.
func WithContextLogger() GetComponentLoggerOption {
	return func(options *GetComponentLoggerOptions) {
		options.ctxLogger = true
	}
}

// Component return a logger with name option base on a logger.
//
// The logger will be lazy set up,Using the global logger by default.
func Component(name string, fields ...zap.Field) ComponentLogger {
	compoenetMu.Lock()
	defer compoenetMu.Unlock()
	if cData, ok := components[name]; ok {
		return cData
	}
	c := &component{
		name:          name,
		builtInFields: append(fields, zap.String(ComponentKey, name)),
		useGlobal:     true,
	}
	c.SetLogger(global)
	components[name] = c
	return c
}

// Logger return component's logger
func (c *component) Logger(opts ...GetComponentLoggerOption) *Logger {
	if c.logger == nil {
		c.SetLogger(global)
	}
	options := GetComponentLoggerOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	if options.ctxLogger {
		return c.cl
	}
	if options.originalLogger {
		return c.logger
	}
	return c.l
}

// SetLogger replace logger, by default component use global logger
func (c *component) SetLogger(logger *Logger) {
	c.useGlobal = logger == global
	if c.logger != logger {
		c.logger = logger
		c.l = logger.With(c.builtInFields...)
		c.cl = c.l.WithOptions(zap.AddCallerSkip(CallerSkip + 1))
	}
}

func (c *component) Debug(msg string, fields ...zap.Field) {
	c.l.Debug(msg, fields...)
}

func (c *component) Info(msg string, fields ...zap.Field) {
	c.l.Info(msg, fields...)
}

func (c *component) Warn(msg string, fields ...zap.Field) {
	c.l.Warn(msg, fields...)
}

func (c *component) Error(msg string, fields ...zap.Field) {
	c.l.Error(msg, fields...)
}

func (c *component) DPanic(msg string, fields ...zap.Field) {
	c.l.DPanic(msg, fields...)
}

func (c *component) Panic(msg string, fields ...zap.Field) {
	c.l.Panic(msg, fields...)
}

func (c *component) Fatal(msg string, fields ...zap.Field) {
	c.l.Fatal(msg, fields...)
}

// Ctx returns a new logger with the context.
func (c *component) Ctx(ctx context.Context) *LoggerWithCtx {
	lc := NewLoggerWithCtx(ctx, c.cl)
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
	ctx context.Context
	l   *Logger
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
	c.l.contextLogger.LogFields(c.l, ctx, lvl, msg, fields)
}

// NewLoggerWithCtx get a logger with context from pool
func NewLoggerWithCtx(ctx context.Context, l *Logger) *LoggerWithCtx {
	return GetLoggerWithCtx(ctx, l)
}
