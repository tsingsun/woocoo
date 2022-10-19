package otellog

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/test"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"testing"
)

func stringlogger() *log.Logger {
	logdata := &test.StringWriteSyncer{}
	zp := test.NewStringLogger(logdata)
	l := log.New(zp)
	l.SetContextLogger(NewContextZapLogger())
	return l
}

func TestContextZapLogger(t *testing.T) {
	tests := []struct {
		name    string
		logger  *log.Logger
		log     func(ctx context.Context, log *log.Logger)
		require func(t *testing.T, event sdktrace.Event)
	}{
		{
			name: "test",
			logger: func() *log.Logger {
				l := stringlogger()
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
	}
	for _, tt := range tests {
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
	}
}

func requireCodeAttrs(t *testing.T, m map[attribute.Key]attribute.Value) {
	fn, ok := m[semconv.CodeFunctionKey]
	require.True(t, ok)
	require.Contains(t, fn.AsString(), "otellog.TestContextZapLogger")

	file, ok := m[semconv.CodeFilepathKey]
	require.True(t, ok)
	require.Contains(t, file.AsString(), "otellog/logger_test.go")

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
