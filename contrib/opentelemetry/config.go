package opentelemetry

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"os"
	"strings"
	"time"
)

const defaultTracerName = "github.com/tsingsun/woocoo"

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (o optionFunc) apply(c *Config) {
	o(c)
}

type Config struct {
	ServiceName                    string             `json:"serviceName,omitempty" yaml:"serviceName"`
	ServiceNamespace               string             `json:"serviceNamespace,omitempty" yaml:"serviceNamespace"`
	ServiceVersion                 string             `json:"serviceVersion,omitempty" yaml:"serviceVersion"`
	Resource                       *resource.Resource `json:"resource,omitempty" yaml:"resource"`
	AttributesEnvKeys              string             `json:"attributesEnvKeys,omitempty" yaml:"attributesEnvKeys"`
	TraceExporterEndpoint          string             `json:"traceExporterEndpoint" yaml:"traceExporterEndpoint"`
	TraceExporterEndpointInsecure  bool               `json:"traceExporterEndpointInsecure" yaml:"traceExporterEndpointInsecure"`
	MetricExporterEndpoint         string             `json:"metricExporterEndpoint" yaml:"metricExporterEndpoint"`
	MetricExporterEndpointInsecure bool               `json:"metricExporterEndpointInsecure" yaml:"metricExporterEndpointInsecure"`
	MetricReportingPeriod          time.Duration      `json:"metricReportingPeriod" yaml:"metricReportingPeriod"`

	TracerProvider trace.TracerProvider
	Tracer         trace.Tracer
	MeterProvider  metric.MeterProvider
	Meter          metric.Meter

	Propagator         propagation.TextMapPropagator
	resourceAttributes map[string]string
}

func NewConfig(name string, opts ...Option) *Config {
	cfg := &Config{
		ServiceName:           name,
		TracerProvider:        otel.GetTracerProvider(),
		MeterProvider:         metric.NewNoopMeterProvider(),
		Propagator:            otel.GetTextMapPropagator(),
		MetricReportingPeriod: time.Second * 30,
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	if name == "" {
		name = defaultTracerName
	}
	cfg.Tracer = cfg.TracerProvider.Tracer(name,
		trace.WithInstrumentationVersion(SemVersion()),
	)
	cfg.Meter = cfg.MeterProvider.Meter(name)

	return cfg
}

// Apply implement conf.Configurable interface
//
// if ServiceName and ServiceVersion and ServiceNameSpace is set in cfg, they will override before
func (c *Config) Apply(cfg *conf.Configuration, path string) {
	c.ServiceName = cfg.Root().AppName()
	c.ServiceVersion = cfg.Root().Version()
	if c.Resource == nil {
		c.Resource = getDefaultResource(c)
	}

	if err := cfg.Unmarshal(&c); err != nil {
		panic(err)
	}
	c.parseEnvKeys()
	if err := c.mergeResource(); err != nil {
		panic(err)
	}
	// trace
	switch c.TraceExporterEndpoint {
	case "stdout":
		if _, err := NewStdTracer(c, stdouttrace.WithPrettyPrint()); err != nil {
			log.Fatalf("%s: %v", "failed to create stdout tracer provider", err)
		}
	case "":
		//tracer do not need
	default:
		if _, err := NewOtlpTracer(c); err != nil {
			log.Fatalf("%s: %v", "failed to create otlp tracer provider", err)
		}
	}
	//metric
	switch c.MetricExporterEndpoint {
	case "stdout":
		if _, err := NewStdMetric(c); err != nil {
			log.Fatalf("%s: %v", "failed to create stdout metric provider", err)
		}
	case "":

	default:
		if _, err := NewOtlpMetric(c); err != nil {
			log.Fatalf("%s: %v", "failed to create otlp metric provider", err)
		}
	}
}

func (c *Config) parseEnvKeys() {
	if c.AttributesEnvKeys == "" {
		return
	}
	envKeys := strings.Split(c.AttributesEnvKeys, "|")
	for _, key := range envKeys {
		key = strings.TrimSpace(key)
		value := os.Getenv(key)
		if value != "" {
			c.resourceAttributes[key] = value
		}
	}
}

// getDefaultResource return a local runtime info
func getDefaultResource(c *Config) *resource.Resource {
	hostname, _ := os.Hostname()
	resource.Environment()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.ServiceName),
		semconv.HostNameKey.String(hostname),
		semconv.ServiceNamespaceKey.String(c.ServiceNamespace),
		semconv.ServiceVersionKey.String(c.ServiceVersion),
		semconv.ProcessPIDKey.Int(os.Getpid()),
		semconv.ProcessCommandKey.String(os.Args[0]),
	)
}

func (c *Config) mergeResource() error {
	var err error
	if c.Resource, err = resource.Merge(getDefaultResource(c), c.Resource); err != nil {
		return err
	}

	r := resource.Environment()
	if c.Resource, err = resource.Merge(c.Resource, r); err != nil {
		return err
	}

	var keyValues []attribute.KeyValue
	for key, value := range c.resourceAttributes {
		keyValues = append(keyValues, attribute.KeyValue{
			Key:   attribute.Key(key),
			Value: attribute.StringValue(value),
		})
	}
	newResource := resource.NewWithAttributes(semconv.SchemaURL, keyValues...)
	if c.Resource, err = resource.Merge(c.Resource, newResource); err != nil {
		return err
	}
	return nil
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
//
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *Config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
//
// If none is specified, the metric.NewNoopMeterProvider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *Config) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
	})
}

// WithPropagator specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *Config) {
		if propagator != nil {
			cfg.Propagator = propagator
		}
	})
}

// WithResource configures attributes on the resource
func WithResource(resource *resource.Resource) Option {
	return optionFunc(func(c *Config) {
		c.Resource = resource
	})
}

// WithResourceAttributes configures attributes on the resource
// 配置上传附加的一些tag信息，例如环境、可用区等
func WithResourceAttributes(attributes map[string]string) Option {
	return optionFunc(func(c *Config) {
		c.resourceAttributes = attributes
	})
}
