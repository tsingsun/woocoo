package otellog

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test/logtest"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
	"time"
)

func stringlogger() (*log.Logger, *logtest.Buffer) {
	logdata := &logtest.Buffer{}
	zp := logtest.NewBuffLogger(logdata, zap.AddStacktrace(zap.DebugLevel), zap.AddCallerSkip(4), zap.AddCaller())
	l := log.New(zp)
	l.SetContextLogger(NewContextZapLogger())
	return l, logdata
}

func TestContextZapLogger(t *testing.T) {
	tests := []struct {
		name    string
		logger  *log.Logger
		log     func(ctx context.Context, log *log.Logger)
		require func(t *testing.T, event sdktrace.Event)
	}{
		{
			name: "info",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Info("hello")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			name: "Warn",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Warn("hello", zap.String("foo", "bar"))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, "bar", foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			name: "Error",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				err := errors.New("some error")
				log.Ctx(ctx).Error("hello", zap.Error(err))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev := m[logSeverityKey]
				assert.Equal(t, "ERROR", sev.AsString())

				msg := m[logMessageKey]
				assert.Equal(t, "hello", msg.AsString())

				excTyp := m[semconv.ExceptionTypeKey]
				assert.Equal(t, "*errors.errorString", excTyp.AsString())

				excMsg := m[semconv.ExceptionMessageKey]
				assert.Equal(t, "some error", excMsg.AsString())

				stack := m[semconv.ExceptionStacktraceKey]
				assert.Contains(t, stack.AsString(), "otellog/otellog_test.go")
				requireCodeAttrs(t, m)
			},
		},
		{
			name: "bools",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Info("hello",
					zap.Bool("b1", true), zap.Bools("b2", []bool{true, false}),
				)
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)
				b1 := m["b1"]
				assert.Equal(t, true, b1.AsBool())
				b2 := m["b2"]
				assert.Equal(t, "[true,false]", b2.AsString())
			},
		},
		{
			name: "numbers",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Info("hello",
					zap.Int("i1", 1), zap.Ints("i2", []int{1, 2}), zap.Uint8("u1", 1),
					zap.Uint16("u2", 1), zap.Uint32("u3", 1), zap.Uint64("u4", 1),
					zap.Float32("f1", float32(1.1)), zap.Float64("f2", float64(1.1)),
					zap.Complex64("c1", complex64(1.1)), zap.Complex128("c2", complex128(1.1)),
				)
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)
				i1 := m["i1"]
				assert.Equal(t, int64(1), i1.AsInt64())
				i2 := m["i2"]
				assert.Equal(t, "[1,2]", i2.AsString())
				u1 := m["u1"]
				assert.Equal(t, int64(1), u1.AsInt64())

				f1 := m["f1"]
				assert.Equal(t, fmt.Sprintf("%f", 1.1), fmt.Sprintf("%f", f1.AsFloat64()))
				f2 := m["f2"]
				assert.Equal(t, fmt.Sprintf("%f", 1.1), fmt.Sprintf("%f", f2.AsFloat64()))

				c1 := m["c1"]
				assert.Contains(t, c1.AsString(), "1.1")
				c2 := m["c2"]
				assert.Contains(t, c2.AsString(), "1.1")
			},
		},
		{
			name: "strings,bytes",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Info("hello",
					zap.String("s1", "1"), zap.Strings("s2", []string{"1", "2"}),
					zap.ByteString("b1", []byte("1")), zap.ByteStrings("b2", [][]byte{[]byte("1"), []byte("2")}),
					zap.Binary("bin1", []byte{97, 65}), zap.Stringer("str1", time.Saturday),
				)
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)
				s1 := m["s1"]
				assert.Equal(t, "1", s1.AsString())
				s2 := m["s2"]
				assert.Equal(t, "[1,2]", s2.AsString())
				b1 := m["b1"]
				assert.Equal(t, "1", b1.AsString())
				b2 := m["b2"]
				assert.Equal(t, "[1,2]", b2.AsString())
				bin1 := m["bin1"]
				assert.Equal(t, "aA", bin1.AsString())
				str1 := m["str1"]
				assert.Equal(t, "Saturday", str1.AsString())
			},
		},
		{
			name: "time",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Info("hello",
					zap.Time("t1", time.Unix(0, 0).UTC()),
					zap.Duration("d1", time.Second),
					zap.Time("t2", time.Date(1676, 1, 1, 0, 0, 0, 0, time.UTC)),
				)
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)
				t1 := m["t1"]
				assert.Equal(t, int64(0), t1.AsInt64())
				d1 := m["d1"]
				assert.Equal(t, int64(time.Second), d1.AsInt64())
				t2 := m["t2"]
				assert.Equal(t, "1676/01/01 00:00:00.000 +00:00", t2.AsString())
			},
		},
		{
			name: "any",
			logger: func() *log.Logger {
				l, _ := stringlogger()
				return l
			}(),
			log: func(ctx context.Context, log *log.Logger) {
				log.Ctx(ctx).Info("hello",
					zap.Object("a1", zapcore.ObjectMarshalerFunc(func(encoder zapcore.ObjectEncoder) error {
						encoder.AddString("foo", "bar")
						encoder.AddInt("baz", 1)
						return nil
					})),
					zap.Reflect("r1", struct {
						Foo string
						Baz int
					}{Foo: "bar", Baz: 1}),
				)
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)
				a1 := m["a1"]
				assert.Equal(t, `{foo=bar,baz=1}`, a1.AsString())
				r1 := m["r1"]
				assert.Equal(t, `{"Foo":"bar","Baz":1}`, r1.AsString())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer("test")

			ctx := context.Background()
			ctx, span := tracer.Start(ctx, "main")

			tt.log(ctx, tt.logger)
			span.End()

			spans := sr.Ended()
			require.Equal(t, 1, len(spans))

			events := spans[0].Events()
			require.Equal(t, 1, len(events))

			event := events[0]
			require.Equal(t, "log", event.Name)
			tt.require(t, event)
		})
	}
}

func requireCodeAttrs(t *testing.T, m map[attribute.Key]attribute.Value) {
	fn, ok := m[semconv.CodeFunctionKey]
	require.True(t, ok)
	require.Contains(t, fn.AsString(), "otellog.TestContextZapLogger")

	file, ok := m[semconv.CodeFilepathKey]
	require.True(t, ok)
	require.Contains(t, file.AsString(), "otellog/otellog_test.go")

	_, ok = m[semconv.CodeLineNumberKey]
	require.True(t, ok)
}

func attrMap(attrs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value, len(attrs))
	for _, kv := range attrs {
		m[kv.Key] = kv.Value
	}
	return m
}
