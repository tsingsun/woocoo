package opentelemetry

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"os"
	"time"
)

// NewStdTracer return a stdout tracer provider and an error if any
func NewStdTracer(c *Config, opts ...stdouttrace.Option) (
	tp *sdktrace.TracerProvider, shutdown func(ctx context.Context) error, err error) {
	exporter, err := stdouttrace.New(opts...)
	if err != nil {
		return
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(c.Resource),
		sdktrace.WithBatcher(exporter),
	)
	shutdown = tp.Shutdown
	return
}

// NewStdMetric return a stdout metric provider and an error if any
func NewStdMetric(c *Config) (mp metric.MeterProvider, shutdown func(ctx context.Context) error, err error) {
	// Print with a JSON encoder that indents with two spaces.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	exporter, err := stdoutmetric.New(stdoutmetric.WithEncoder(enc))
	if err != nil {
		return
	}
	shutdown = exporter.Shutdown
	mp, err = initMetric(c, exporter)
	return
}

func initMetric(c *Config, exporter sdkmetric.Exporter) (metric.MeterProvider, error) {
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(c.Resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(c.MetricPeriodicReaderInterval))),
	)
	// Golang runtime
	err := runtime.Start(runtime.WithMeterProvider(meterProvider), runtime.WithMinimumReadMemStatsInterval(time.Second))
	return meterProvider, err
}

func NewOtlpTracer(c *Config) (tp trace.TracerProvider, shutdown func(ctx context.Context) error, err error) {
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
		return
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(c.Resource),
		sdktrace.WithSpanProcessor(bsp),
	)
	shutdown = exporter.Shutdown
	return
}

func NewOtlpMetric(c *Config) (mp metric.MeterProvider, shutdown func(ctx context.Context) error, err error) {
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
		return
	}
	shutdown = exporter.Shutdown
	mp, err = initMetric(c, exporter)
	return
}

// NewTraceInOption return trace provider which export has been set in `trace.TracerProviderOption`
func NewTraceInOption(c *Config) (tp trace.TracerProvider, shutdown func(ctx context.Context) error, err error) {
	if len(c.tops) > 0 {
		df := []sdktrace.TracerProviderOption{sdktrace.WithResource(c.Resource)}
		c.tops = append(df, c.tops...)
		tpt := sdktrace.NewTracerProvider(c.tops...)
		tp = tpt
		shutdown = tpt.Shutdown
	}
	return
}

// NewMetricInOption return meter which export has been set in `metric.Option`
func NewMetricInOption(c *Config) (mp metric.MeterProvider, shutdown func(ctx context.Context) error, err error) {
	if len(c.mops) > 0 {
		df := []sdkmetric.Option{sdkmetric.WithResource(c.Resource)}
		c.mops = append(df, c.mops...)
		mpt := sdkmetric.NewMeterProvider(c.mops...)
		c.MeterProvider = mp
		mp = mpt
		shutdown = mpt.Shutdown
	}
	return
}
