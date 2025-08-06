package telemetry

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

// fakeTracer 用于测试
type fakeTracer struct {
	trace.Tracer
}

func (f *fakeTracer) tracer() {
	//TODO implement me
	panic("implement me")
}

func (f *fakeTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.GetTracerProvider().Tracer("fake").Start(ctx, spanName, opts...)
}

func TestInject(t *testing.T) {
	oriConfig := globalConfig
	oriTextMapPropagator := otel.GetTextMapPropagator()
	defer func() {
		globalConfig = oriConfig
		if globalConfig != nil {
			otel.SetTextMapPropagator(globalConfig.TextMapPropagator)
		} else {
			otel.SetTextMapPropagator(oriTextMapPropagator)
		}
	}()
	t.Run("inject returns nil when tracer is nil", func(t *testing.T) {
		if globalConfig != nil {
			globalConfig = nil
		}
		carrier := Inject(context.Background())
		assert.Nil(t, carrier)
		globalConfig = &Config{
			Tracer: nil,
		}
		carrier = Inject(context.Background())
		assert.Nil(t, carrier)
	})

	t.Run("inject returns carrier when tracer is set", func(t *testing.T) {
		globalConfig = &Config{
			TextMapPropagator: b3.New(),
			Tracer:            &fakeTracer{},
		}
		otel.SetTextMapPropagator(globalConfig.TextMapPropagator)
		ctx, _ := globalConfig.Tracer.Start(context.Background(), "fake")
		carrier := Inject(ctx)
		assert.Equal(t, carrier.Get("b3"), "0")
	})
}

func TestStartWith(t *testing.T) {
	oriConfig := globalConfig
	oriTextMapPropagator := otel.GetTextMapPropagator()
	defer func() {
		globalConfig = oriConfig
		if globalConfig != nil {
			otel.SetTextMapPropagator(globalConfig.TextMapPropagator)
		} else {
			otel.SetTextMapPropagator(oriTextMapPropagator)
		}
	}()

	t.Run("returns ctx and nil when globalConfig or tracer is nil", func(t *testing.T) {
		globalConfig = nil
		ctx := context.Background()
		newCtx, span := StartWith(ctx, "test", nil)
		assert.Equal(t, ctx, newCtx)
		assert.Nil(t, span)

		globalConfig = &Config{Tracer: nil}
		newCtx, span = StartWith(ctx, "test", nil)
		assert.Equal(t, ctx, newCtx)
		assert.Nil(t, span)
	})

	t.Run("extracts context from carrier and starts span", func(t *testing.T) {
		globalConfig = &Config{
			TextMapPropagator: b3.New(),
			Tracer:            &fakeTracer{},
		}
		otel.SetTextMapPropagator(globalConfig.TextMapPropagator)
		ctx := context.Background()
		carrier := propagation.MapCarrier{}
		carrier.Set("b3", "0")
		newCtx, span := StartWith(ctx, "test-span", carrier)
		assert.NotNil(t, span)
		assert.NotNil(t, newCtx)
	})

	t.Run("starts span without carrier", func(t *testing.T) {
		globalConfig = &Config{
			TextMapPropagator: b3.New(),
			Tracer:            &fakeTracer{},
		}
		otel.SetTextMapPropagator(globalConfig.TextMapPropagator)
		ctx := context.Background()
		newCtx, span := StartWith(ctx, "test-span", nil)
		assert.NotNil(t, span)
		assert.NotNil(t, newCtx)
	})
}
