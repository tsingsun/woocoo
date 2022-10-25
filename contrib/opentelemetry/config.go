package opentelemetry

import (
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultTracerName = "github.com/tsingsun/woocoo"
)

var (
	globalConfig *Config
)

func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}

func GlobalConfig() *Config {
	return globalConfig
}

func GlobalTracer() trace.Tracer {
	return globalConfig.Tracer
}

func GlobalMeter() metric.Meter {
	return globalConfig.Meter
}

func GetTextMapPropagator() propagation.TextMapPropagator {
	return globalConfig.Propagator
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (o optionFunc) apply(c *Config) {
	o(c)
}

// Config is the configuration for the opentelemetry instrumentation,Through it to set global tracer and meter provider.
type Config struct {
	ServiceName                    string `json:"serviceName,omitempty" yaml:"serviceName"`
	ServiceNamespace               string `json:"serviceNamespace,omitempty" yaml:"serviceNamespace"`
	ServiceVersion                 string `json:"serviceVersion,omitempty" yaml:"serviceVersion"`
	AttributesEnvKeys              string `json:"attributesEnvKeys,omitempty" yaml:"attributesEnvKeys"`
	TraceExporterEndpoint          string `json:"traceExporterEndpoint" yaml:"traceExporterEndpoint"`
	TraceExporterEndpointInsecure  bool   `json:"traceExporterEndpointInsecure" yaml:"traceExporterEndpointInsecure"`
	MetricExporterEndpoint         string `json:"metricExporterEndpoint" yaml:"metricExporterEndpoint"`
	MetricExporterEndpointInsecure bool   `json:"metricExporterEndpointInsecure" yaml:"metricExporterEndpointInsecure"`
	// the intervening time between exports for a PeriodicReader.
	MetricPeriodicReaderInterval time.Duration `json:"metricReportingPeriod" yaml:"metricReportingPeriod"`

	Resource       *resource.Resource            `json:"-" yaml:"-"`
	TracerProvider trace.TracerProvider          `json:"-" yaml:"-"`
	Tracer         trace.Tracer                  `json:"-" yaml:"-"`
	MeterProvider  metric.MeterProvider          `json:"-" yaml:"-"`
	Meter          metric.Meter                  `json:"-" yaml:"-"`
	Propagator     propagation.TextMapPropagator `json:"-" yaml:"-"`

	resourceAttributes map[string]string
	// with options
	mops []sdkmetric.Option
	tops []sdktrace.TracerProviderOption

	shutdowns []func(ctx context.Context) error
	asGlobal  bool
}

func NewConfig(cnf *conf.Configuration, opts ...Option) *Config {
	c := &Config{
		ServiceName:                  cnf.Root().AppName(),
		MetricPeriodicReaderInterval: time.Second * 30,
		asGlobal:                     true,
		resourceAttributes:           make(map[string]string),
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	if c.ServiceName == "" {
		c.ServiceName = defaultTracerName
	}
	c.Apply(cnf)
	return c
}

// Apply implement conf.Configurable interface
//
// if ServiceName and ServiceVersion and ServiceNameSpace is set in cfg, they will override before
func (c *Config) Apply(cnf *conf.Configuration) {
	c.ServiceName = cnf.Root().AppName()
	c.ServiceVersion = cnf.Root().Version()
	if c.Resource == nil {
		c.Resource = getDefaultResource(c)
	}

	if err := cnf.Unmarshal(&c); err != nil {
		panic(err)
	}
	c.parseEnvKeys()
	if err := c.mergeResource(); err != nil {
		panic(err)
	}

	c.applyTracerProvider()
	c.applyMetricProvider()
	if c.TracerProvider != nil {
		otel.SetTracerProvider(c.TracerProvider)
	} else {
		c.TracerProvider = otel.GetTracerProvider()
	}
	c.Tracer = c.TracerProvider.Tracer(c.ServiceName, trace.WithInstrumentationVersion(SemVersion()))

	if c.MeterProvider != nil {
		global.SetMeterProvider(c.MeterProvider)
	} else {
		c.MeterProvider = global.MeterProvider()
	}
	c.Meter = c.MeterProvider.Meter(c.ServiceName)

	if c.Propagator != nil {
		otel.SetTextMapPropagator(c.Propagator)
	} else {
		c.Propagator = otel.GetTextMapPropagator()
	}

	if globalConfig == nil {
		SetGlobalConfig(c)
	}
}

func (c *Config) applyTracerProvider() {
	var (
		shutdown func(ctx context.Context) error
		err      error
	)
	// trace
	switch c.TraceExporterEndpoint {
	case "otlp":
		c.TracerProvider, shutdown, err = NewOtlpTracer(c)
	case "stdout":
		c.TracerProvider, shutdown, err = NewStdTracer(c, stdouttrace.WithPrettyPrint())
	default:
		c.TracerProvider, shutdown, err = NewTraceInOption(c)
	}
	if err != nil {
		panic(fmt.Errorf("failed to create %s tracer provider:%v", c.TraceExporterEndpoint, err))
	}
	if shutdown != nil {
		c.shutdowns = append(c.shutdowns, shutdown)
	}
}

func (c *Config) applyMetricProvider() {
	var (
		shutdown func(ctx context.Context) error
		err      error
	)
	//metric
	switch c.MetricExporterEndpoint {
	case "otlp":
		c.MeterProvider, shutdown, err = NewOtlpMetric(c)
	case "stdout":
		c.MeterProvider, shutdown, err = NewStdMetric(c)
	default:
		c.MeterProvider, shutdown, err = NewMetricInOption(c)
	}
	if err != nil {
		panic(fmt.Errorf("failed to create %s metric provider:%v", c.MetricExporterEndpoint, err))
	}
	if shutdown != nil {
		c.shutdowns = append(c.shutdowns, shutdown)
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

func (c *Config) Shutdown() {
	var wg sync.WaitGroup
	wg.Add(len(c.shutdowns))
	for _, shutdown := range c.shutdowns {
		go func(shutdown func(ctx context.Context) error) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			if err := shutdown(ctx); err != nil {
				log.Errorf("Error shutting down tracer provider: %v", err)
			}
		}(shutdown)
	}
	wg.Wait()
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
//
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider, shutdown func(ctx context.Context) error) Option {
	return optionFunc(func(cfg *Config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
		if shutdown != nil {
			cfg.shutdowns = append(cfg.shutdowns, shutdown)
		}
	})
}

// WithTracerProviderOptions specifies sdk tracer provider options to use for creating a tracer.
func WithTracerProviderOptions(opts ...sdktrace.TracerProviderOption) Option {
	return optionFunc(func(cfg *Config) {
		cfg.tops = opts
	})
}

// WithMeterProviderOptions specifies sdk tracer provider options to use for creating a tracer.
func WithMeterProviderOptions(opts ...sdkmetric.Option) Option {
	return optionFunc(func(cfg *Config) {
		cfg.mops = opts
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
//
// If none is specified, the metric.NewNoopMeterProvider is used.
func WithMeterProvider(provider metric.MeterProvider, shutdown func(ctx context.Context) error) Option {
	return optionFunc(func(cfg *Config) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
		if shutdown != nil {
			cfg.shutdowns = append(cfg.shutdowns, shutdown)
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
//
// example: zone=shanghai|app=app1|app_version=1.0.0
func WithResourceAttributes(attributes map[string]string) Option {
	return optionFunc(func(c *Config) {
		c.resourceAttributes = attributes
	})
}
