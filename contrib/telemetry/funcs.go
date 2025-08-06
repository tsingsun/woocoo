package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"reflect"
)

// Attribute returns an attribute.KeyValue from a key and any value.
func Attribute(key string, value any) attribute.KeyValue {
	switch value := value.(type) {
	case nil:
		return attribute.String(key, "<nil>")
	case string:
		return attribute.String(key, value)
	case int:
		return attribute.Int(key, value)
	case int64:
		return attribute.Int64(key, value)
	case uint64:
		return attribute.Int64(key, int64(value))
	case float64:
		return attribute.Float64(key, value)
	case bool:
		return attribute.Bool(key, value)
	case fmt.Stringer:
		return attribute.String(key, value.String())
	}

	rv := reflect.ValueOf(value)

	switch rv.Kind() {
	case reflect.Array:
		rv2 := reflect.New(rv.Type()).Elem()
		rv2.Set(rv)
		rv = rv2.Slice(0, rv.Len())
		fallthrough
	case reflect.Slice:
		switch reflect.TypeOf(value).Elem().Kind() {
		case reflect.Bool:
			return attribute.BoolSlice(key, rv.Interface().([]bool))
		case reflect.Int:
			return attribute.IntSlice(key, rv.Interface().([]int))
		case reflect.Int64:
			return attribute.Int64Slice(key, rv.Interface().([]int64))
		case reflect.Float64:
			return attribute.Float64Slice(key, rv.Interface().([]float64))
		case reflect.String:
			return attribute.StringSlice(key, rv.Interface().([]string))
		default:
			return attribute.KeyValue{Key: attribute.Key(key)}
		}
	case reflect.Bool:
		return attribute.Bool(key, rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return attribute.Int64(key, rv.Int())
	case reflect.Float64:
		return attribute.Float64(key, rv.Float())
	case reflect.String:
		return attribute.String(key, rv.String())
	}
	if b, err := json.Marshal(value); b != nil && err == nil {
		return attribute.String(key, string(b))
	}
	return attribute.String(key, fmt.Sprint(value))
}

// Inject set cross-cutting concerns from the Context into the carrier.
func Inject(ctx context.Context) propagation.TextMapCarrier {
	if globalConfig == nil || globalConfig.Tracer == nil {
		return nil
	}
	// 将跟踪信息注入消息
	propagator := globalConfig.TextMapPropagator
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	return carrier
}

// StartWith starts a span and extracts the carrier info to the parameter context.
// Note: parameter 'ctx' is the parent context need extract.
func StartWith(ctx context.Context, spanName string, carrier propagation.TextMapCarrier, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if globalConfig == nil || globalConfig.Tracer == nil {
		return ctx, nil
	}
	if carrier != nil {
		propagator := globalConfig.TextMapPropagator
		ctx = propagator.Extract(ctx, carrier)
	}
	ctx, span := globalConfig.Tracer.Start(ctx, spanName, opts...)
	return ctx, span
}
