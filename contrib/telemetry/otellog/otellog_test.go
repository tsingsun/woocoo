package otellog

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/testco/logtest"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.uber.org/zap"
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

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				excTyp, ok := m[semconv.ExceptionTypeKey]
				require.True(t, ok)
				require.Equal(t, "*errors.errorString", excTyp.AsString())

				excMsg, ok := m[semconv.ExceptionMessageKey]
				require.True(t, ok)
				require.Equal(t, "some error", excMsg.AsString())

				stack, ok := m[semconv.ExceptionStacktraceKey]
				require.True(t, ok)
				require.Contains(t, stack.AsString(), "otellog/otellog_test.go")
				requireCodeAttrs(t, m)
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

func TestBufferArrayEncoder(t *testing.T) {
	tests := []struct {
		name     string
		input    []any
		do       func(*bufferArrayEncoder, ...any)
		expected []string
		batch    bool
	}{
		{
			name:  "int",
			input: []any{1, int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1)},
			do: func(enc *bufferArrayEncoder, input ...any) {
				funcs := []func(enc *bufferArrayEncoder, v any){
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendInt(v.(int))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendInt8(v.(int8))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendInt16(v.(int16))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendInt32(v.(int32))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendInt64(v.(int64))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendUint(v.(uint))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendUint8(v.(uint8))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendUint16(v.(uint16))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendUint32(v.(uint32))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendUint64(v.(uint64))
					},
				}
				for i, v := range input {
					enc := &bufferArrayEncoder{}
					funcs[i](enc, v)
					assert.Equal(t, []string{"1"}, enc.stringsSlice)
				}
			},
			batch: true,
		},
		{
			name:  "string",
			input: []any{"hello"},
			do: func(enc *bufferArrayEncoder, input ...any) {
				enc.AppendString("hello")
			},
			expected: []string{"hello"},
		},
		{
			name:  "bytes",
			input: []any{[]byte("hello")},
			do: func(enc *bufferArrayEncoder, input ...any) {
				enc.AppendByteString([]byte("hello"))
			},
			expected: []string{"[104 101 108 108 111]"},
		},
		{
			name:  "struct",
			input: []any{struct{ A int }{1}},
			do: func(enc *bufferArrayEncoder, input ...any) {
				assert.NoError(t, enc.AppendReflected(input[0]))
			},
			expected: []string{"{1}"},
		},
		{
			name:  "slice",
			input: []any{[]int{1, 2, 3}},
			do: func(encoder *bufferArrayEncoder, input ...any) {
				assert.NoError(t, encoder.AppendReflected(input[0]))
			},
			expected: []string{"[1 2 3]"},
		},
		{
			name:  "array",
			input: []any{[3]int{1, 2, 3}},
			do: func(encoder *bufferArrayEncoder, input ...any) {
				assert.NoError(t, encoder.AppendReflected(input[0]))
			},
			expected: []string{"[1 2 3]"},
		},
		{
			name:  "map",
			input: []any{map[string]int{"a": 1, "b": 2}},
			do: func(encoder *bufferArrayEncoder, input ...any) {
				assert.NoError(t, encoder.AppendReflected(input[0]))
			},
			expected: []string{"map[a:1 b:2]"},
		},
		{
			name:  "bool",
			input: []any{true},
			do: func(encoder *bufferArrayEncoder, input ...any) {
				encoder.AppendBool(input[0].(bool))
			},
			expected: []string{"true"},
		},
		{
			name:  "float",
			input: []any{1.1, float32(1.1)},
			do: func(enc *bufferArrayEncoder, input ...any) {
				funcs := []func(enc *bufferArrayEncoder, v any){
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendFloat64(v.(float64))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendFloat32(v.(float32))
					},
				}
				for i, v := range input {
					enc := &bufferArrayEncoder{}
					funcs[i](enc, v)
					assert.Equal(t, []string{"1.1"}, enc.stringsSlice)
				}
			},
			batch: true,
		},
		{
			name:  "time",
			input: []any{time.Unix(0, 0).UTC()},
			do: func(encoder *bufferArrayEncoder, input ...any) {
				encoder.AppendTime(input[0].(time.Time))
			},
			expected: []string{"1970-01-01 00:00:00 +0000 UTC"},
		},
		{
			name:  "duration",
			input: []any{time.Second},
			do: func(encoder *bufferArrayEncoder, input ...any) {
				encoder.AppendDuration(input[0].(time.Duration))
			},
			expected: []string{"1s"},
		},
		{
			name:  "complex",
			input: []any{complex64(complex(1, 2)), complex128(complex(1, 2))},
			do: func(enc *bufferArrayEncoder, input ...any) {
				funcs := []func(enc *bufferArrayEncoder, v any){
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendComplex64(v.(complex64))
					},
					func(enc *bufferArrayEncoder, v any) {
						enc.AppendComplex128(v.(complex128))
					},
				}
				for i, v := range input {
					enc := &bufferArrayEncoder{}
					funcs[i](enc, v)
					assert.Equal(t, []string{"(1+2i)"}, enc.stringsSlice)
				}
			},
			batch: true,
		},
		{
			name:  "uintptr",
			input: []any{uintptr(1)},
			do: func(enc *bufferArrayEncoder, input ...any) {
				enc.AppendUintptr(input[0].(uintptr))
			},
			expected: []string{"1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.batch {
				tt.do(&bufferArrayEncoder{}, tt.input...)
				return
			}
			enc := &bufferArrayEncoder{}
			tt.do(enc, tt.input[0])
			assert.Equal(t, tt.expected, enc.stringsSlice)
		})
	}
}
