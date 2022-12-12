package telemetry

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/zipkin"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"os"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	require.NoError(t, os.Setenv("WOOCOO_TEST_NAME", "woocoo"))
	type args struct {
		cnf  *conf.Configuration
		opts []Option
	}
	tests := []struct {
		name string
		args args
		want *Config
	}{
		{
			name: "config",
			args: args{
				cnf: conf.NewFromStringMap(map[string]interface{}{
					"appName": "test",
					"otel": map[string]interface{}{
						"traceExporter":     "stdout",
						"metricExporter":    "stdout",
						"attributesEnvKeys": "WOOCOO_TEST_NAME|NOEXISTS",
					},
				}).Sub("otel"),
			},
			want: &Config{
				ServiceName:                  "test",
				MetricPeriodicReaderInterval: time.Second * 30,
				MetricExporterEndpoint:       "stdout",
				TraceExporterEndpoint:        "stdout",
				AttributesEnvKeys:            "WOOCOO_TEST_NAME|NOEXISTS",
				resourceAttributes:           map[string]string{"WOOCOO_TEST_NAME": "woocoo"},
				Resource: resource.NewSchemaless(
					attribute.String("WOOCOO_TEST_NAME", "woocoo"),
				),
			},
		},
		{
			name: "with",
			args: args{
				cnf: conf.NewFromStringMap(map[string]interface{}{
					"appName": "test-with",
					"otel": map[string]interface{}{
						"traceExporterEndpoint": "",
					},
				}).Sub("otel"),
				opts: func() (opts []Option) {
					opts = append(opts,
						WithTracerProvider(zipkinProvider(t)), WithPropagator(b3.New()),
						WithMeterProvider(prometheusProvider(t)),
						WithResourceAttributes(map[string]string{"test": "test"}),
						WithResource(resource.NewSchemaless(Attribute("attr1", "attr1"))),
					)
					return
				}(),
			},
			want: &Config{
				ServiceName:                  "test-with",
				MetricPeriodicReaderInterval: time.Second * 30,
				TraceExporterEndpoint:        "",
				resourceAttributes: map[string]string{
					"test": "test",
				},
				Resource: resource.NewSchemaless(
					Attribute("test", "test"),
					Attribute("attr1", "attr1"),
				),
			},
		},
		{
			name: "with provider options",
			args: args{
				cnf: conf.NewFromStringMap(map[string]interface{}{
					"appName": "test-with",
					"otel": map[string]interface{}{
						"traceExporter": "",
					},
				}).Sub("otel"),
				opts: func() (opts []Option) {
					opts = append(opts,
						WithResourceAttributes(map[string]string{"test": "test"}),
						WithResource(resource.NewSchemaless(Attribute("attr1", "attr1"))),
						WithTracerProviderOptions(zipkinProviderOptions(t)...), WithPropagator(b3.New()),
						WithMeterProviderOptions(prometheusProviderOptions(t)...),
					)
					return
				}(),
			},
			want: &Config{
				ServiceName:                  "test-with",
				MetricPeriodicReaderInterval: time.Second * 30,
				TraceExporterEndpoint:        "",
				resourceAttributes: map[string]string{
					"test": "test",
				},
				Resource: resource.NewSchemaless(
					Attribute("test", "test"),
					Attribute("attr1", "attr1"),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConfig(tt.args.cnf, tt.args.opts...)
			assert.Equal(t, tt.want.ServiceName, got.ServiceName)
			assert.Equal(t, tt.want.MetricPeriodicReaderInterval, got.MetricPeriodicReaderInterval)
			assert.Equal(t, tt.want.MetricExporterEndpoint, got.MetricExporterEndpoint)
			assert.Equal(t, tt.want.TraceExporterEndpoint, got.TraceExporterEndpoint)
			assert.Equal(t, tt.want.AttributesEnvKeys, got.AttributesEnvKeys)
			assert.Equal(t, tt.want.resourceAttributes, got.resourceAttributes)
			assert.Subset(t, got.Resource.Attributes(), tt.want.Resource.Attributes())
		})
	}
}

func zipkinProviderOptions(t *testing.T) []sdktrace.TracerProviderOption {
	// Create a Zipkin exporter
	exporter, err := zipkin.New("")
	require.NoError(t, err)
	return []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exporter),
	}
}

func zipkinProvider(t *testing.T) (*sdktrace.TracerProvider, func(ctx context.Context) error) {
	// Create a Zipkin exporter
	exporter, err := zipkin.New("")
	require.NoError(t, err)

	// Create a trace provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	return tp, exporter.Shutdown
}

func prometheusProviderOptions(t *testing.T) []sdkmetric.Option {
	exporter, err := prometheus.New()
	require.NoError(t, err)
	return []sdkmetric.Option{
		sdkmetric.WithReader(exporter),
	}
}

func prometheusProvider(t *testing.T) (*sdkmetric.MeterProvider, func(ctx context.Context) error) {
	exporter, err := prometheus.New()
	require.NoError(t, err)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter))
	return mp, exporter.Shutdown
}

func TestOtlp(t *testing.T) {
	type args struct {
		cnf *conf.Configuration
	}
	tests := []struct {
		name string
		args args
		do   func(t *testing.T, got *Config)
	}{
		{
			name: "otlp",
			args: args{
				cnf: conf.NewFromStringMap(map[string]interface{}{
					"appName": "test",
					"otel": map[string]any{
						"traceExporter":         "otlp",
						"traceExporterEndpoint": "127.0.0.1:4317",
						"client": map[string]any{
							"grpcDialOption": []any{
								map[string]any{"insecure": nil},
								map[string]any{"block": nil},
								map[string]any{"timeout": "5s"},
							},
						},
					},
				}).Sub("otel"),
			},
			do: func(t *testing.T, cfg *Config) {
				tracer := otel.GetTracerProvider().Tracer("woocoo-otlp-test")
				_, span := tracer.Start(context.Background(), "woocoo-otlp-test")
				defer span.End()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConfig(tt.args.cnf)
			tt.do(t, got)
			got.Shutdown()
		})
	}
}
