package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
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
	return globalConfig.TextMapPropagator
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
//
// Propagator could be not set or b3. b3 is simple init without any option.
type Config struct {
	cnf               *conf.Configuration
	ServiceName       string `json:"serviceName,omitempty" yaml:"serviceName"`
	ServiceNamespace  string `json:"serviceNamespace,omitempty" yaml:"serviceNamespace"`
	ServiceVersion    string `json:"serviceVersion,omitempty" yaml:"serviceVersion"`
	AttributesEnvKeys string `json:"attributesEnvKeys,omitempty" yaml:"attributesEnvKeys"`
	TraceExporter     string `json:"traceExporter,omitempty" yaml:"traceExporter,omitempty"`
	MetricExporter    string `json:"metricExporter,omitempty" yaml:"metricExporter,omitempty"`
	// the intervening time between exports for a PeriodicReader.
	MetricPeriodicReaderInterval time.Duration                 `json:"metricReportingPeriod" yaml:"metricReportingPeriod"`
	Propagator                   string                        `json:"propagator,omitempty" yaml:"propagator,omitempty"`
	Resource                     *resource.Resource            `json:"-" yaml:"-"`
	TracerProvider               trace.TracerProvider          `json:"-" yaml:"-"`
	Tracer                       trace.Tracer                  `json:"-" yaml:"-"`
	MeterProvider                metric.MeterProvider          `json:"-" yaml:"-"`
	Meter                        metric.Meter                  `json:"-" yaml:"-"`
	TextMapPropagator            propagation.TextMapPropagator `json:"-" yaml:"-"`
	Headers                      map[string]string             `json:"headers,omitempty" yaml:"headers,omitempty"`

	resourceAttributes map[string]string
	// with options
	mops []sdkmetric.Option
	tops []sdktrace.TracerProviderOption

	shutdowns []func(ctx context.Context) error
	asGlobal  bool
}

func NewConfig(cnf *conf.Configuration, opts ...Option) *Config {
	c := &Config{
		cnf:                          cnf,
		MetricPeriodicReaderInterval: time.Second * 30,
		Headers:                      make(map[string]string),
		asGlobal:                     true,
		resourceAttributes:           make(map[string]string),
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	c.Apply(cnf)
	return c
}

// Apply implement conf.Configurable interface
//
// if ServiceName and ServiceVersion and ServiceNameSpace is set in cfg, they will override before
func (c *Config) Apply(cnf *conf.Configuration) {
	if err := cnf.Unmarshal(&c); err != nil {
		panic(err)
	}
	if c.ServiceName == "" {
		c.ServiceName = cnf.Root().AppName()
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = cnf.Root().Version()
	}
	if c.ServiceNamespace == "" {
		c.ServiceNamespace = cnf.Root().Namespace()
	}
	if c.Resource == nil {
		c.Resource = getDefaultResource(c)
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
		otel.SetMeterProvider(c.MeterProvider)
	} else {
		c.MeterProvider = otel.GetMeterProvider()
	}
	c.Meter = c.MeterProvider.Meter(c.ServiceName)

	if c.Propagator == "b3" {
		c.TextMapPropagator = b3.New()
	}
	if c.TextMapPropagator != nil {
		otel.SetTextMapPropagator(c.TextMapPropagator)
	} else {
		c.TextMapPropagator = otel.GetTextMapPropagator()
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
	switch c.TraceExporter {
	case "otlp":
		c.TracerProvider, shutdown, err = NewOtlpTracer(c)
	case "stdout":
		c.TracerProvider, shutdown, err = NewStdTracer(c, stdouttrace.WithPrettyPrint())
	default:
		if c.TracerProvider == nil && len(c.tops) > 0 {
			c.TracerProvider, shutdown, err = NewTraceInOption(c)
		}
	}
	if err != nil {
		panic(fmt.Errorf("failed to create %s tracer provider:%v", c.TraceExporter, err))
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
	switch c.MetricExporter {
	case "otlp":
		c.MeterProvider, shutdown, err = NewOtlpMetric(c)
	case "stdout":
		c.MeterProvider, shutdown, err = NewStdMetric(c)
	default:
		if c.MeterProvider == nil && len(c.mops) > 0 {
			c.MeterProvider, shutdown, err = NewMetricInOption(c)
		}
	}
	if err != nil {
		panic(fmt.Errorf("failed to create %s metric provider:%v", c.MetricExporter, err))
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

func WithName(name string) Option {
	return optionFunc(func(c *Config) {
		c.ServiceName = name
	})
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
			cfg.TextMapPropagator = propagator
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
