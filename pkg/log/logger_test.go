package log

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strconv"
	"strings"
	"testing"
	"time"
)

type user int

func (u user) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if int(u) < 0 {
		return errors.New("too few users")
	}
	enc.AddInt("users", int(u))
	return nil
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

func initStringWriteSyncer(opts ...zap.Option) *test.StringWriteSyncer {
	opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	logdata := &test.StringWriteSyncer{}
	zl := test.NewStringLogger(logdata, opts...)
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
				conf.NewFromStringMap(map[string]interface{}{}).AsGlobal()
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
				Logger:            tt.fields.Logger,
				WithTraceID:       tt.fields.WithTraceID,
				DisableCaller:     tt.fields.DisableCaller,
				DisableStacktrace: tt.fields.DisableStacktrace,
				contextLogger:     tt.fields.contextLogger,
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
				Logger:            tt.fields.Logger,
				WithTraceID:       tt.fields.WithTraceID,
				DisableCaller:     tt.fields.DisableCaller,
				DisableStacktrace: tt.fields.DisableStacktrace,
				contextLogger:     tt.fields.contextLogger,
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
			fields: fields{
				WithTraceID:       false,
				DisableCaller:     false,
				DisableStacktrace: false,
				contextLogger:     &DefaultContextLogger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logdata := &test.StringWriteSyncer{}
			zp := tt.fields.Logger
			if zp == nil {
				zp = test.NewStringLogger(logdata)
			}
			l := &Logger{
				Logger:            zp,
				WithTraceID:       tt.fields.WithTraceID,
				DisableCaller:     tt.fields.DisableCaller,
				DisableStacktrace: tt.fields.DisableStacktrace,
				contextLogger:     tt.fields.contextLogger,
			}
			got := l.Ctx(tt.args.ctx)
			got.Debug("debug", zap.String("key", "value"))
			got.Info("info", zap.String("key", "value"))
			got.Warn("warn", zap.String("key", "value"))
			got.Error("error", zap.String("key", "value"))
			got.DPanic("dpanic", zap.String("key", "value"))
			got.Log(zap.ErrorLevel, "log", []zap.Field{zap.String("key", "value")})
			assert.Len(t, logdata.Entry, 6)
		})
	}
}

func TestLoggerLog(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name    string
		args    args
		do      func() *test.StringWriteSyncer
		require func(*test.StringWriteSyncer)
		panic   bool
	}{
		{
			name: "all but panic",
			do: func() *test.StringWriteSyncer {
				applyGlobal(true)
				logdata := initStringWriteSyncer()
				Debug("debug", "x")
				Info("infox")
				Warn("warnx")
				Error("errorx")
				DPanic("dpanicx")
				return logdata
			},
			require: func(logdata *test.StringWriteSyncer) {
				ss := logdata
				assert.Contains(t, ss.Entry[0], "debugx")
				assert.Contains(t, ss.Entry[1], "infox")
				assert.Contains(t, ss.Entry[2], "warnx")
				assert.Contains(t, ss.Entry[3], "errorx")
				if !assert.Contains(t, strings.Split(ss.Entry[3], "\\n\\t")[1], "log/logger_test.go") {
					t.Log(global)
				}
				assert.Contains(t, ss.Entry[4], "dpanicx")
				assert.Contains(t, strings.Split(ss.Entry[4], "\\n\\t")[1], "log/logger_test.go")
			},
		},
		{
			name: "panic error",
			do: func() *test.StringWriteSyncer {
				applyGlobal(true)
				logdata := initStringWriteSyncer()
				Panic("error", zap.Error(errors.New("panicx")))
				return logdata
			},
			require: func(logdata *test.StringWriteSyncer) {
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
			do: func() *test.StringWriteSyncer {
				applyGlobal(true)
				logdata := initStringWriteSyncer()
				Debugf("debug%s", "x")
				Infof("info%s", "x")
				Warnf("warn%s", "x")
				Errorf("error%s", "x")
				DPanicf("dpanic%s", "x")
				return logdata
			},
			require: func(logdata *test.StringWriteSyncer) {
				ss := logdata
				assert.Contains(t, ss.Entry[0], "debugx")
				assert.Contains(t, ss.Entry[1], "infox")
				assert.Contains(t, ss.Entry[2], "warnx")
				assert.Contains(t, ss.Entry[3], "errorx")
				if !assert.Contains(t, strings.Split(ss.Entry[3], "\\n\\t")[1], "log/logger_test.go") {
					t.Log(global)
				}
				assert.Contains(t, ss.Entry[4], "dpanicx")
				assert.Contains(t, strings.Split(ss.Entry[4], "\\n\\t")[1], "log/logger_test.go")
			},
		},
		{
			name: "panic format error",
			do: func() *test.StringWriteSyncer {
				applyGlobal(true)
				logdata := initStringWriteSyncer()
				Panicf("error%s", "x")
				return logdata
			},
			require: func(logdata *test.StringWriteSyncer) {
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

func TestLogger_Component(t *testing.T) {
	type fields struct {
		logger ComponentLogger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "component-1", fields: fields{logger: Component("component-1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logdata := &test.StringWriteSyncer{}
			zp := test.NewStringLogger(logdata)
			l := &Logger{
				Logger: zp,
			}
			got := tt.fields.logger
			got.SetLogger(l)
			got.Debug("debug", zap.String("key", "value"))
			got.Info("info", zap.String("key", "value"))
			got.Warn("warn", zap.String("key", "value"))
			got.Error("error", zap.String("key", "value"))
			got.DPanic("dpanic", zap.String("key", "value"))
			assert.Len(t, logdata.Entry, 5)
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
	logger.Info("info for object", zap.Object("object", user(0)))
	logger.Info("info for Binary", zap.Binary("binary", []byte{1, 2, 3, 4, 5}))
	logger.Info("info for array", zap.Bools("array", []bool{true, false}))
}

func TestLogger_callSkip(t *testing.T) {
	tests := []struct {
		name    string
		do      func() *test.StringWriteSyncer
		require func(data *test.StringWriteSyncer)
	}{
		{
			name: "global",
			do: func() *test.StringWriteSyncer {
				applyGlobal(true)
				data := initStringWriteSyncer()
				Error("errorx")
				return data
			},
			require: func(data *test.StringWriteSyncer) {
				ss := data.Entry[0]
				assert.Contains(t, ss, "errorx")
				st := strings.Split(ss, "\\n\\t")[1]
				assert.Contains(t, st, "log/logger_test.go")
			},
		},
		{
			name: "ctx",
			do: func() *test.StringWriteSyncer {
				applyGlobal(true)
				data := initStringWriteSyncer()
				Global().Ctx(context.Background()).Error("errorx")
				return data
			},
			require: func(data *test.StringWriteSyncer) {
				ss := data.Entry[0]
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
			require.Len(t, data.Entry, 1)
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
