package otelweb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	otelwoocoo "github.com/tsingsun/woocoo/contrib/telemetry"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelName  = "otel"
	tracerKey = "otel-go-contrib-tracer"
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/tsingsun/woocoo/otelweb"
)

// Middleware returns middleware that will trace incoming requests.
type Middleware struct {
	cfg *otelwoocoo.Config
	// RecordRequestBody whether to record request body
	// not recommend to enable it in production environment
	RecordRequestBody bool `yaml:"recordRequestBody" json:"recordRequestBody"`
	// MaxBodySize limit the max size of request body to read, default is 4096
	MaxBodySize int `yaml:"maxBodySize" json:"maxBodySize"`
}

// New see handler.MiddlewareNewFunc
func New() handler.Middleware {
	return &Middleware{
		RecordRequestBody: false,
		MaxBodySize:       4096,
	}
}

// RegisterMiddleware return a web Option for otel middleware
func RegisterMiddleware() web.Option {
	return web.WithMiddlewareNewFunc(otelName, New)
}

func (h *Middleware) Name() string {
	return otelName
}

func (h *Middleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	h.cfg = otelwoocoo.GlobalConfig()
	if err := cfg.Unmarshal(h); err != nil {
		panic(fmt.Errorf("unmarshal otelweb middleware config error: %w", err))
	}
	return middleware(h)
}

// Shutdown will flush the tracer's span processor and then shut it down.
//
// the middleware uses the global tracer provider, so this function is empty.you should call otelwoocoo.Shutdown() when
// application shutdown.
func (h *Middleware) Shutdown(_ context.Context) error {
	return nil
}

// middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func middleware(h *Middleware) gin.HandlerFunc {
	prop := h.cfg.TextMapPropagator
	tracer := h.cfg.Tracer
	return func(c *gin.Context) {
		c.Set(tracerKey, tracer)
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()
		ctx := prop.Extract(savedCtx, propagation.HeaderCarrier(c.Request.Header))
		opts := []trace.SpanStartOption{
			trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Request)...),
			trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Request)...),
			trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(c.Request.Host, c.FullPath(), c.Request)...),
			trace.WithSpanKind(trace.SpanKindServer),
		}
		spanName := c.FullPath()
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		}
		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()
		if h.RecordRequestBody && c.Request.Body != nil && c.Request.Body != http.NoBody {
			var body []byte
			body, _ = io.ReadAll(c.Request.Body)
			if len(body) > h.MaxBodySize {
				span.SetAttributes(attribute.String("http.request.body", string(body[:h.MaxBodySize])))
			} else {
				span.SetAttributes(attribute.String("http.request.body", string(body)))

			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)
		// serve the request to the next middleware
		c.Next()

		status := c.Writer.Status()
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(status)
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			for _, e := range c.Errors {
				span.RecordError(e.Err)
			}
		}
	}
}
