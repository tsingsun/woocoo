package telemetry

import (
	"context"
	"encoding/json"
	"github.com/tsingsun/woocoo/rpc/grpcx"
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
	traceCfg := c.cnf.Sub(c.TraceExporter)
	conn, err := grpcx.NewClient(traceCfg).Dial(traceCfg.String("endpoint"),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
	)
	if err != nil {
		return
	}

	ctx := context.Background()
	// Set up a trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
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
	metricCfg := c.cnf.Sub(c.MetricExporter)
	gclient := grpcx.NewClient(metricCfg)
	conn, err := gclient.Dial(metricCfg.String("endpoint"),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
	)
	if err != nil {
		return
	}
	exporter, err := otlpmetricgrpc.New(context.Background(),
		otlpmetricgrpc.WithGRPCConn(conn),
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
	df := []sdktrace.TracerProviderOption{sdktrace.WithResource(c.Resource)}
	c.tops = append(df, c.tops...)
	tpt := sdktrace.NewTracerProvider(c.tops...)
	tp = tpt
	shutdown = tpt.Shutdown
	return
}

// NewMetricInOption return meter which export has been set in `metric.Option`
func NewMetricInOption(c *Config) (mp metric.MeterProvider, shutdown func(ctx context.Context) error, err error) {
	df := []sdkmetric.Option{sdkmetric.WithResource(c.Resource)}
	c.mops = append(df, c.mops...)
	mpt := sdkmetric.NewMeterProvider(c.mops...)
	c.MeterProvider = mp
	mp = mpt
	shutdown = mpt.Shutdown
	return
}
