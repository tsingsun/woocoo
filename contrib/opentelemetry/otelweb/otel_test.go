package otelweb

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/contrib/opentelemetry"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"testing"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
)

func TestOtelHandler(t *testing.T) {
	var cfgStr = `
appName: test
otel:
  traceExporterEndpoint: stdout
  metricExporterEndpoint: stdout
`
	rcfg := conf.NewFromBytes([]byte(cfgStr))
	hcfg := rcfg.Sub("otel")

	tests := []struct {
		name string
	}{
		{name: "base"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New().ApplyFunc(hcfg)
			assert.NotNil(t, got)
		})
	}
}

func TestGetSpanNotInstrumented(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {
		// Assert we don't have a span on the context.
		span := trace.SpanFromContext(c.Request.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		_, _ = c.Writer.Write([]byte("ok"))
	})
	r := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := trace.NewNoopTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(tracerName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := gin.New()

	otelcfg := opentelemetry.NewConfig("foobar", opentelemetry.WithTracerProvider(provider))

	router.Use(middleware(otelcfg))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := trace.NewNoopTracerProvider()
	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(tracerName).Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := gin.New()

	otelcfg := opentelemetry.NewConfig("foobar", opentelemetry.WithTracerProvider(provider), opentelemetry.WithPropagator(b3))

	router.Use(middleware(otelcfg))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}
