package otelweb

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelwoocoo "github.com/tsingsun/woocoo/contrib/telemetry"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"net/http"
	"net/http/httptest"
	"testing"
)

func init() {
	gin.SetMode(gin.ReleaseMode) // silence annoying log msgs
}

func TestMiddleware_NewConfig(t *testing.T) {
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
						"traceExporter":  "stdout",
						"metricExporter": "stdout",
					},
				}),
			},
			handlerFunc: func(c *gin.Context) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otlecfg := otelwoocoo.NewConfig(tt.args.cfg.Sub("otel"))
			defer otlecfg.Shutdown()
			h := New().(*Middleware)
			assert.Equal(t, "otel", h.Name())
			router := gin.New()
			router.Use(h.ApplyFunc(tt.args.cfg))
			router.GET("/ping", tt.handlerFunc)
			r := httptest.NewRequest("GET", "/ping", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			assert.NoError(t, h.Shutdown(context.Background()))

			wcweb := web.New(RegisterMiddleware())
			wcweb.Router().GET("/ping", tt.handlerFunc)
			wcweb.Router().ServeHTTP(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGetSpanNotInstrumented(t *testing.T) {
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
	provider := noop.NewTracerProvider()
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
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
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
	provider := noop.NewTracerProvider()
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
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
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

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelwoocoo.SetGlobalConfig(nil)
	cnf := conf.NewFromStringMap(map[string]any{
		"appName": "childSpan",
	})
	sr := tracetest.NewSpanRecorder()
	otelcfg := otelwoocoo.NewConfig(cnf, otelwoocoo.WithTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)), nil))
	defer otelcfg.Shutdown()

	router := gin.New()
	router.Use(middleware(otelcfg))

	router.GET("/user/:id", func(c *gin.Context) {})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestError(t *testing.T) {
	otelwoocoo.SetGlobalConfig(nil)
	cnf := conf.NewFromStringMap(map[string]any{
		"appName": "testError",
	})
	sr := tracetest.NewSpanRecorder()
	otelcfg := otelwoocoo.NewConfig(cnf, otelwoocoo.WithTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)), nil))
	defer otelcfg.Shutdown()
	// setup
	router := gin.New()
	router.Use(middleware(otelcfg))

	// configure a handler that returns an error and 5xx status
	// code
	router.GET("/server_err", func(c *gin.Context) {
		_ = c.Error(errors.New("oh no one"))
		_ = c.AbortWithError(http.StatusInternalServerError, errors.New("oh no two"))
	})
	r := httptest.NewRequest("GET", "/server_err", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "example.com"))
	assert.Contains(t, attr, attribute.Int("http.status_code", http.StatusInternalServerError))

	// verify the error events
	events := span.Events()
	require.Len(t, events, 2)
	assert.Equal(t, "exception", events[0].Name)
	assert.Contains(t, events[0].Attributes, attribute.String("exception.type", "*errors.errorString"))
	assert.Contains(t, events[0].Attributes, attribute.String("exception.message", "oh no one"))
	assert.Equal(t, "exception", events[1].Name)
	assert.Contains(t, events[1].Attributes, attribute.String("exception.type", "*errors.errorString"))
	assert.Contains(t, events[1].Attributes, attribute.String("exception.message", "oh no two"))

	// server errors set the status
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, "Error #01: oh no one\nError #02: oh no two\n", span.Status().Description)
}
