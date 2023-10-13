package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/logtest"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type user struct {
	Name, Email string
	CreatedAt   time.Time
}

func (u *user) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", u.Name)
	enc.AddString("email", u.Email)
	enc.AddInt64("createdAt", u.CreatedAt.UnixNano())
	return nil
}

type users []*user

func (uu users) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	var err error
	for i := range uu {
		err = multierr.Append(err, arr.AppendObject(uu[i]))
	}
	return err
}

type tracerContextLogger struct{}

func (n *tracerContextLogger) LogFields(log *Logger, _ context.Context, lvl zapcore.Level, msg string, fields []zap.Field) {
	if log.WithTraceID {
		fields = append(fields, zap.String(log.TraceIDKey, "123456"))
	}
	log.Log(lvl, msg, fields...)
}

func applyGlobal(disableStacktrace bool) {
	glog := InitGlobalLogger()
	glog.Apply(conf.NewFromBytes([]byte(fmt.Sprintf(`
disableTimestamp: false
disableErrorVerbose: false
cores:
- level: debug
  disableCaller: true
  disableStacktrace: %s`, strconv.FormatBool(disableStacktrace)))))
	glog.AsGlobal()
}

func initBuffWriteSyncer(opts ...zap.Option) *logtest.Buffer {
	opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	logdata := &logtest.Buffer{}
	zl := logtest.NewBuffLogger(logdata, opts...)
	glog := Global().Logger()
	opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zl.Core()
	}))
	glog.WithOptions(opts...).AsGlobal()
	return logdata
}

func TestNewBuiltIn(t *testing.T) {
	tests := []struct {
		name    string
		want    func() *Logger
		wantErr bool
	}{
		{
			name: "miss config",
			want: func() *Logger {
				conf.NewFromStringMap(map[string]any{}).AsGlobal()
				return nil
			},
			wantErr: true,
		},
		{
			name: "global",
			want: func() *Logger {
				conf.New(conf.WithLocalPath(testdata.TestConfigFile()), conf.WithBaseDir(testdata.BaseDir())).Load().AsGlobal()
				return Global().Logger()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want()
			if tt.wantErr {
				assert.Panics(t, func() {
					NewBuiltIn()
				})
				return
			}
			got := NewBuiltIn()
			assert.Same(t, global, got)
			assert.NotNil(t, global)
		})
	}
}

func TestLogger_AsGlobal(t *testing.T) {
	type fields struct {
		Logger            *zap.Logger
		WithTraceID       bool
		DisableCaller     bool
		DisableStacktrace bool
		contextLogger     ContextLogger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			fields: fields{
				Logger:      zap.NewNop(),
				WithTraceID: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{
				Logger:        tt.fields.Logger,
				WithTraceID:   tt.fields.WithTraceID,
				contextLogger: tt.fields.contextLogger,
			}
			Component("test").SetLogger(New(zap.NewNop()))
			got := l.AsGlobal()
			// pointer not equal
			require.Same(t, got, global)
			assert.NotSame(t, got, Component("test").Logger())
			for name, i2 := range components {
				if i2.Logger() == got {
					require.Same(t, got, i2.Logger(), name)
				}
			}
		})
	}
}

func TestLogger_WithOptions(t *testing.T) {
	type fields struct {
		Logger            *zap.Logger
		WithTraceID       bool
		DisableCaller     bool
		DisableStacktrace bool
		contextLogger     ContextLogger
	}
	type args struct {
		opts []zap.Option
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Logger
	}{
		{
			fields: fields{
				Logger:            zap.NewNop(),
				WithTraceID:       true,
				DisableCaller:     false,
				DisableStacktrace: false,
				contextLogger:     nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{
				Logger:        tt.fields.Logger,
				WithTraceID:   tt.fields.WithTraceID,
				contextLogger: tt.fields.contextLogger,
			}
			got := l.WithOptions(tt.args.opts...)
			require.NotSame(t, got, l)
			require.NotSame(t, got.Operator(), l.Operator())
			assert.Equalf(t, l, got, "WithOptions(%v)", tt.args.opts)
		})
	}
}

func TestLogger_Ctx(t *testing.T) {
	type fields struct {
		Logger            *zap.Logger
		WithTraceID       bool
		TraceIDKey        string
		DisableCaller     bool
		DisableStacktrace bool
		contextLogger     ContextLogger
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *LoggerWithCtx
	}{
		{
			name: "with ctx",
			fields: fields{
				WithTraceID:       false,
				DisableCaller:     false,
				DisableStacktrace: false,
				contextLogger:     &DefaultContextLogger{},
			},
		},
		{
			name: "with ctx and trace id",
			fields: fields{
				WithTraceID:       true,
				TraceIDKey:        "traceId",
				DisableCaller:     false,
				DisableStacktrace: false,
				contextLogger:     &tracerContextLogger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logdata := &logtest.Buffer{}
			zp := tt.fields.Logger
			if zp == nil {
				zp = logtest.NewBuffLogger(logdata)
			}
			l := &Logger{
				Logger:        zp,
				WithTraceID:   tt.fields.WithTraceID,
				TraceIDKey:    tt.fields.TraceIDKey,
				contextLogger: tt.fields.contextLogger,
			}
			assert.NotNil(t, l.ContextLogger())
			l.Ctx(tt.args.ctx).Debug("debug", zap.String("key", "value"))
			l.Ctx(tt.args.ctx).Info("info", zap.String("key", "value"))
			l.Ctx(tt.args.ctx).Warn("warn", zap.String("key", "value"))
			l.Ctx(tt.args.ctx).Error("error", zap.String("key", "value"))
			l.Ctx(tt.args.ctx).DPanic("dpanic", zap.String("key", "value"))
			l.Ctx(tt.args.ctx).Log(zap.ErrorLevel, "log", []zap.Field{zap.String("key", "value")})
			l.Ctx(tt.args.ctx).WithOptions(zap.AddCaller()).Info("addcaller", zap.String("key", "value"))
			assert.Len(t, logdata.Lines(), 7)
			if _, ok := l.contextLogger.(*tracerContextLogger); ok {
				assert.Contains(t, logdata.LastLine(), l.TraceIDKey)
			} else {
				assert.Contains(t, logdata.LastLine(), "log/logger.go")
			}
		})
	}
}

func TestLoggerLog(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name    string
		args    args
		do      func() *logtest.Buffer
		require func(buffer *logtest.Buffer)
		panic   bool
	}{
		{
			name: "all but panic",
			do: func() *logtest.Buffer {
				applyGlobal(true)
				logdata := initBuffWriteSyncer()
				Debug("debug", "x")
				Info("infox")
				Warn("warnx")
				Error("errorx")
				DPanic("dpanicx")
				return logdata
			},
			require: func(logdata *logtest.Buffer) {
				ss := logdata
				lines := ss.Lines()
				assert.Contains(t, lines[0], "debugx")
				assert.Contains(t, lines[1], "infox")
				assert.Contains(t, lines[2], "warnx")
				assert.Contains(t, lines[3], "errorx")
				if !assert.Contains(t, strings.Split(lines[3], "\\n\\t")[1], "log/logger_test.go") {
					t.Log(global)
				}
				assert.Contains(t, lines[4], "dpanicx")
				assert.Contains(t, strings.Split(lines[4], "\\n\\t")[1], "log/logger_test.go")
			},
		},
		{
			name: "panic error",
			do: func() *logtest.Buffer {
				applyGlobal(true)
				logdata := initBuffWriteSyncer()
				Panic("error", zap.Error(errors.New("panicx")))
				return logdata
			},
			require: func(logdata *logtest.Buffer) {
				ss := logdata
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "public error")
			},
			panic: true,
		},
		{
			name: "all format but panic",
			do: func() *logtest.Buffer {
				applyGlobal(true)
				logdata := initBuffWriteSyncer()
				Debugf("debug%s", "x")
				Infof("info%s", "x")
				Warnf("warn%s", "x")
				Errorf("error%s", "x")
				DPanicf("dpanic%s", "x")
				return logdata
			},
			require: func(logdata *logtest.Buffer) {
				ss := logdata
				lines := ss.Lines()
				assert.Contains(t, lines[0], "debugx")
				assert.Contains(t, lines[1], "infox")
				assert.Contains(t, lines[2], "warnx")
				assert.Contains(t, lines[3], "errorx")
				if !assert.Contains(t, strings.Split(lines[3], "\\n\\t")[1], "log/logger_test.go") {
					t.Log(global)
				}
				assert.Contains(t, lines[4], "dpanicx")
				assert.Contains(t, strings.Split(lines[4], "\\n\\t")[1], "log/logger_test.go")
			},
		},
		{
			name: "panic format error",
			do: func() *logtest.Buffer {
				applyGlobal(true)
				initBuffWriteSyncer()
				Panicf("error%s", "x")
				return nil
			},
			require: func(logdata *logtest.Buffer) {
				ss := logdata
				all := ss.String()
				// panic
				assert.Contains(t, all, "panic")
				assert.Contains(t, all, "request")
				assert.Contains(t, all, "stacktrace")
				assert.Contains(t, all, "public error")
			},
			panic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				assert.Panics(t, func() {
					tt.require(tt.do())
				})
			} else {
				tt.require(tt.do())
			}
		})
	}
}

func TestLogger_TextEncode(t *testing.T) {
	var cfgStr = `
development: true
log:
  disableTimestamp: false
  disableErrorVerbose: false
  cores:
    - level: debug
      disableCaller: false
      disableStacktrace: false
      encoding: text
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).Load()
	got, err := NewConfig(cfg.Sub("log"))
	assert.NoError(t, err)
	zl, err := got.BuildZap()
	assert.NoError(t, err)
	logger := New(zl)
	logger.Info("info")
	// TODO: github action bug for output error
	logger.Info("info for scalar", zap.String("string", "it's a string"),
		zap.Int("int", 1), zap.Int8("int8", 1), zap.Int16("int16", 1), zap.Int32("int32", 1), zap.Int64("int64", 1),
		zap.Uint("uint", 1), zap.Uint8("uint8", 1), zap.Uint16("uint16", 1), zap.Uint32("uint32", 1), zap.Uint64("uint64", 1),
		zap.Float64("float64", 64.0), zap.Float32("float32", float32(32.0)), zap.Bool("bool", true),
		zap.Duration("duration", 1), zap.Time("time", time.Now()),
		zap.ByteString("byteString", []byte("byteString\n\r\t")),
		zap.Complex64("complex64", 1), zap.Complex128("complex128", 1),
	)
	logger.Info("info for object", zap.Any("any", testdata.TestStruct()))
	logger.Info("info for object", zap.Object("object", &user{Name: "user"}))
	logger.Info("info for Binary", zap.Binary("binary", []byte{1, 2, 3, 4, 5}))
	logger.Info("info for array", zap.Bools("array", []bool{true, false}))
	logger.Info("info for dict", zap.Dict("dict", zap.String("key", "value"), zap.String("key1", "value")))
}

func TestLogger_callSkip(t *testing.T) {
	tests := []struct {
		name    string
		do      func() *logtest.Buffer
		require func(data *logtest.Buffer)
	}{
		{
			name: "global",
			do: func() *logtest.Buffer {
				applyGlobal(true)
				data := initBuffWriteSyncer()
				Error("errorx")
				return data
			},
			require: func(data *logtest.Buffer) {
				ss := data.String()
				assert.Contains(t, ss, "errorx")
				st := strings.Split(ss, "\\n\\t")[1]
				assert.Contains(t, st, "log/logger_test.go")
			},
		},
		{
			name: "ctx",
			do: func() *logtest.Buffer {
				applyGlobal(true)
				data := initBuffWriteSyncer()
				Global().Ctx(context.Background()).Error("errorx")
				return data
			},
			require: func(data *logtest.Buffer) {
				ss := data.Lines()[0]
				assert.Contains(t, ss, "errorx")
				st := strings.Split(ss, "\\n\\t")[1]
				assert.Contains(t, st, "log/logger_test.go")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.do()
			tt.require(data)
			require.Len(t, data.Lines(), 1)
		})
	}
}

// -----------------------------------------------------------------------------
func TestPrintLogo(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "print-logo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintLogo()
		})
	}
}

func TestLogger_IOWriter(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		logdata := &logtest.Buffer{}
		zp := logtest.NewBuffLogger(logdata)
		l := &Logger{
			Logger: zp,
		}
		golog := log.New(io.Discard, "", log.LstdFlags)
		golog.SetOutput(l.IOWriter(zapcore.DebugLevel))
		golog.Println("standard log")
		assert.Contains(t, logdata.String(), "standard log")
	})
	t.Run("with-level", func(t *testing.T) {
		logdata := &logtest.Buffer{}
		zp := logtest.NewBuffLogger(logdata)
		l := &Logger{
			Logger: zp.WithOptions(zap.IncreaseLevel(zapcore.InfoLevel)),
		}
		golog := log.New(io.Discard, "", log.LstdFlags)
		golog.SetOutput(l.IOWriter(zapcore.DebugLevel))
		golog.Println("[DEBUG] Debugging")
		assert.NotContains(t, logdata.String(), "Debugging")
		golog.Print("[WARN] Warning")
		assert.Contains(t, logdata.String(), "Warning")
	})
}
