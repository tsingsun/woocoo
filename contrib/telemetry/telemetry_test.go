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
	"strings"
	"testing"
	"time"
)

func TestAttribute(t *testing.T) {
	type aliasF64 float64
	type aliasS string
	type aliasB bool
	tests := []struct {
		value any
		want  attribute.KeyValue
	}{
		{nil, attribute.String("key", "<nil>")},
		{string("value"), attribute.String("key", "value")},
		{aliasS("value"), attribute.String("key", "value")},
		{int(1), attribute.Int("key", 1)},
		{int16(1), attribute.Int("key", 1)},
		{int64(2), attribute.Int64("key", 2)},
		{uint64(3), attribute.Int64("key", 3)},
		{float64(1.5), attribute.Float64("key", 1.5)},
		{aliasF64(1.5), attribute.Float64("key", 1.5)},
		{true, attribute.Bool("key", true)},
		{aliasB(true), attribute.Bool("key", true)},
		{time.Saturday, attribute.String("key", "Saturday")},
		{struct{ Name string }{Name: "value"}, attribute.String("key", `{"Name":"value"}`)},
		{[]string{"a", "b"}, attribute.StringSlice("key", []string{"a", "b"})},
		{[2]string{"a", "b"}, attribute.StringSlice("key", []string{"a", "b"})},
		{[2]string{"a", "b"}, attribute.StringSlice("key", []string{"a", "b"})},
		{[]float64{1.5, 2.5}, attribute.Float64Slice("key", []float64{1.5, 2.5})},
		{[]int{1, 2}, attribute.IntSlice("key", []int{1, 2})},
		{[]int64{3, 4}, attribute.Int64Slice("key", []int64{3, 4})},
		{[]bool{true, false}, attribute.BoolSlice("key", []bool{true, false})},
		{complex(1, 2), attribute.String("key", "(1+2i)")},
		// unsupported but should not get error
		{[]int8{1, 2}, attribute.KeyValue{Key: attribute.Key("key")}},
	}
	for _, tt := range tests {
		got := Attribute("key", tt.value)
		assert.Equal(t, tt.want, got)
	}
}

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
						"propagators":       "b3",
					},
				}).Sub("otel"),
			},
			want: &Config{
				ServiceName:                  "test",
				MetricPeriodicReaderInterval: time.Second * 30,
				MetricExporter:               "stdout",
				TraceExporter:                "stdout",
				AttributesEnvKeys:            "WOOCOO_TEST_NAME|NOEXISTS",
				resourceAttributes:           map[string]string{"WOOCOO_TEST_NAME": "woocoo"},
				Resource: resource.NewSchemaless(
					attribute.String("WOOCOO_TEST_NAME", "woocoo"),
				),
				TextMapPropagator: b3.New(),
			},
		},
		{
			name: "with",
			args: args{
				cnf: conf.NewFromStringMap(map[string]interface{}{
					"appName": "test-with",
					"otel": map[string]interface{}{
						"traceExporter": "",
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
				TraceExporter:                "",
				MetricPeriodicReaderInterval: time.Second * 30,
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
				TraceExporter:                "",
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
			assert.Equal(t, tt.want.MetricExporter, got.MetricExporter)
			assert.Equal(t, tt.want.TraceExporter, got.TraceExporter)
			assert.Equal(t, tt.want.AttributesEnvKeys, got.AttributesEnvKeys)
			assert.Equal(t, tt.want.resourceAttributes, got.resourceAttributes)
			assert.Subset(t, got.Resource.Attributes(), tt.want.Resource.Attributes())
			assert.NotNil(t, GlobalConfig())
			assert.NotNil(t, GlobalTracer())
			assert.NotNil(t, GlobalMeter())
			assert.NotNil(t, GetTextMapPropagator())
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
						"traceExporter": "otlp",
						"otlp": map[string]any{
							"endpoint": "127.0.0.1:4317",
							"client": map[string]any{
								"dialOption": []any{
									map[string]any{"tls": nil},
									map[string]any{"block": nil},
									map[string]any{"timeout": "1s"},
								},
							},
						},
						"metricExporter": "otlp",
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
			// server is not ready(github action), so we need to recover panic
			defer func() {
				if r := recover(); r != nil {
					if strings.Contains(r.(error).Error(), "context deadline exceeded") {
						return
					}
					t.Errorf("panic: %v", r)
				}
			}()
			got := NewConfig(tt.args.cnf)
			tt.do(t, got)
			got.Shutdown()
		})
	}
}
