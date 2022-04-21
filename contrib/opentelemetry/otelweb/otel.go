package otelweb

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	otelwoocoo "github.com/tsingsun/woocoo/contrib/opentelemetry"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"sync"
	"time"
)

const (
	tracerKey  = "otel-go-contrib-tracer"
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/tsingsun/web/otelwoocoo"
)

type Handler struct {
	service string
	Config  *otelwoocoo.Config
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Name() string {
	return "otel"
}

func (h *Handler) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	h.Config = otelwoocoo.NewConfig(cfg.Root().AppName())
	h.Config.Apply(cfg, "")
	return h.middleware()
}

func (h *Handler) Shutdown() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if tp, ok := h.Config.TracerProvider.(*sdktrace.TracerProvider); ok {
			if err := tp.Shutdown(ctx); err != nil {
				log.Errorf("Error shutting down tracer provider: %v", err)
			}
		}
		wg.Done()
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if mp, ok := h.Config.MeterProvider.(*controller.Controller); ok {
			if err := mp.Stop(ctx); err != nil {
				log.Errorf("Error shutting down metric provider: %v", err)
			}
		}
		wg.Done()
	}()
	wg.Wait()
}

// middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func (h *Handler) middleware() gin.HandlerFunc {
	cfg := h.Config
	service := h.Config.ServiceName
	return func(c *gin.Context) {
		c.Set(tracerKey, cfg.Tracer)
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()
		ctx := cfg.Propagator.Extract(savedCtx, propagation.HeaderCarrier(c.Request.Header))
		opts := []trace.SpanStartOption{
			trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Request)...),
			trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Request)...),
			trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, c.FullPath(), c.Request)...),
			trace.WithSpanKind(trace.SpanKindServer),
		}
		spanName := c.FullPath()
		if spanName == "" {
			spanName = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		}
		ctx, span := cfg.Tracer.Start(ctx, spanName, opts...)
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
