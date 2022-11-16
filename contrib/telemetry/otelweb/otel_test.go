package otelweb

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	otelwoocoo "github.com/tsingsun/woocoo/contrib/telemetry"
	"github.com/tsingsun/woocoo/pkg/conf"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_NewConfig(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name        string
		args        args
		handlerFunc gin.HandlerFunc
	}{
		{
			name: "std",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"appName": "gin-web",
					"otel": map[string]any{
						"traceExporterEndpoint":  "stdout",
						"metricExporterEndpoint": "stdout",
					},
				}),
			},
			handlerFunc: func(c *gin.Context) {
				return
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otlecfg := otelwoocoo.NewConfig(tt.args.cfg.Sub("otel"))
			defer otlecfg.Shutdown()
			h := NewMiddleware()
			router := gin.New()
			router.Use(h.ApplyFunc(tt.args.cfg))
			router.GET("/ping", tt.handlerFunc)
			r := httptest.NewRequest("GET", "/ping", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
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
	otelwoocoo.SetGlobalConfig(nil)
	provider := trace.NewNoopTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())
	cnf := conf.NewFromStringMap(map[string]any{
		"appName": "foobar",
	})
	otelcfg := otelwoocoo.NewConfig(cnf, otelwoocoo.WithTracerProvider(provider, nil))
	defer otelcfg.Shutdown()
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
	router.Use(middleware(otelcfg))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	otelwoocoo.SetGlobalConfig(nil)
	provider := trace.NewNoopTracerProvider()
	b3 := b3prop.New()
	cnf := conf.NewFromStringMap(map[string]any{
		"appName": "foobar",
	})
	otelcfg := otelwoocoo.NewConfig(cnf, otelwoocoo.WithTracerProvider(provider, nil), otelwoocoo.WithPropagator(b3))
	defer otelcfg.Shutdown()

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
	router.Use(middleware(otelcfg))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}
