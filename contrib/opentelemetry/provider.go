package opentelemetry

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"time"
)

// NewStdTracer return a stdout tracer provider and an error if any
func NewStdTracer(c *Config, opts ...stdouttrace.Option) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(opts...)
	if err != nil {
		return nil, err
		//log.Fatalf("%s: %v", "failed to create stdout tracer provider", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(c.Resource),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

// NewStdMetric return a stdout metric provider and an error if any
func NewStdMetric(c *Config) (metric.MeterProvider, error) {
	exporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		return nil, err
		//log.Fatalf("%s: %v", "failed to create stdout metric provider", err)
	}
	return initMetric(c, exporter)
}

func initMetric(c *Config, exporter export.Exporter) (metric.MeterProvider, error) {
	cont := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(c.MetricReportingPeriod),
		controller.WithResource(c.Resource),
	)
	global.SetMeterProvider(cont)
	if err := cont.Start(context.Background()); err != nil {
		return nil, err
	}

	//// host indicator
	//if err := host.Start(host.WithMeterProvider(cont)); err != nil {
	//	return nil, err
	//}

	// Golang runtime
	err := runtime.Start(runtime.WithMeterProvider(cont), runtime.WithMinimumReadMemStatsInterval(time.Second))
	return cont, err
}

func NewOtlpTracer(c *Config) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()
	traceSecureOption := otlptracegrpc.WithInsecure()
	if c.TraceExporterEndpointInsecure {
		traceSecureOption = otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}

	// Set up a trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		traceSecureOption,
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
		otlptracegrpc.WithEndpoint(c.TraceExporterEndpoint),
		otlptracegrpc.WithCompressor(gzip.Name),
	)
	if err != nil {
		//log.Fatalf("%s: %v", "failed to create trace exporter", err)
		return nil, err
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(c.Resource),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tracerProvider, nil
}

func NewOtlpMetric(c *Config) (metric.MeterProvider, error) {
	secureOption := otlpmetricgrpc.WithInsecure()
	if c.MetricExporterEndpointInsecure {
		secureOption = otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}
	exporter, err := otlpmetricgrpc.New(context.Background(),
		secureOption,
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
		otlpmetricgrpc.WithEndpoint(c.MetricExporterEndpoint),
		otlpmetricgrpc.WithCompressor(gzip.Name),
	)
	if err != nil {
		return nil, err
		//log.Fatalf("%s: %v", "failed to create otlp metric provider", err)
	}
	return initMetric(c, exporter)
}
