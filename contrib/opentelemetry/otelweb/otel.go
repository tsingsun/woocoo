package otelweb

import (
	"fmt"
	"github.com/gin-gonic/gin"
	otelwoocoo "github.com/tsingsun/woocoo/contrib/opentelemetry"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerKey  = "otel-go-contrib-tracer"
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/tsingsun/web/otelwoocoo"
)

// Middleware returns middleware that will trace incoming requests.
type Middleware struct {
	cfg *otelwoocoo.Config
}

func NewMiddleware() *Middleware {
	return &Middleware{}
}

func (h *Middleware) Name() string {
	return "otel"
}

func (h *Middleware) ApplyFunc(_ *conf.Configuration) gin.HandlerFunc {
	h.cfg = otelwoocoo.GlobalConfig()
	return middleware(h.cfg)
}

// Shutdown will flush the tracer's span processor and then shut it down.
//
// the middleware uses the global tracer provider, so this function is empty.you should call otelwoocoo.Shutdown() when
// application shutdown.
func (h *Middleware) Shutdown() {
}

// middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func middleware(cfg *otelwoocoo.Config) gin.HandlerFunc {
	prop := cfg.Propagator
	tracer := cfg.Tracer
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
			span.SetAttributes(attribute.String("web.errors", c.Errors.String()))
		}
	}
}

// HTML will trace the rendering of the template as a child of the
// span in the given context. This is a replacement for
// gin.Context.HTML function - it invokes the original function after
// setting up the span.
func HTML(c *gin.Context, code int, name string, obj interface{}) {
	var tracer trace.Tracer
	tracerInterface, ok := c.Get(tracerKey)
	if ok {
		tracer, ok = tracerInterface.(trace.Tracer)
	}
	if !ok {
		c.HTML(code, name, obj)
		return
	}
	savedContext := c.Request.Context()
	defer func() {
		c.Request = c.Request.WithContext(savedContext)
	}()
	opt := trace.WithAttributes(attribute.String("go.template", name))
	_, span := tracer.Start(savedContext, "gin.renderer.html", opt)
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("error rendering template:%s: %s", name, r)
			span.RecordError(err)
			span.SetStatus(codes.Error, "template failure")
			span.End()
			panic(r)
		} else {
			span.End()
		}
	}()
	c.HTML(code, name, obj)
}
