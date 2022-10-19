package otellog

import (
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/contrib/opentelemetry"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

// trace to the log.XXX call method
const runtimeCallerSkip = 5

var (
	logSeverityKey = attribute.Key("log.severity")
	logMessageKey  = attribute.Key("log.message")
)

type ContextLogger struct {
	callerDepth      int
	errorStatusLevel zapcore.Level
}

func NewContextZapLogger() *ContextLogger {
	return &ContextLogger{}
}

func (l *ContextLogger) LogFields(logger *log.Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return fields
	}

	attrs := make([]attribute.KeyValue, len(fields))
	for _, f := range fields {
		if f.Type == zapcore.NamespaceType {
			continue
		}
		attrs = appendField(attrs, f)
	}
	l.log(logger, span, lvl, msg, attrs)

	if logger.WithTraceID {
		traceID := span.SpanContext().TraceID().String()
		fields = append(fields, zap.String("trace_id", traceID))
	}

	return fields
}

func (l *ContextLogger) log(logger *log.Logger, span trace.Span, lvl zapcore.Level, msg string, attrs []attribute.KeyValue) {
	attrs = append(attrs, logSeverityKey.String(levelString(lvl)))
	attrs = append(attrs, logMessageKey.String(msg))

	if !logger.DisableCaller {
		if fn, file, line, ok := runtimeCaller(runtimeCallerSkip + l.callerDepth); ok {
			if fn != "" {
				attrs = append(attrs, semconv.CodeFunctionKey.String(fn))
			}
			if file != "" {
				attrs = append(attrs, semconv.CodeFilepathKey.String(file))
				attrs = append(attrs, semconv.CodeLineNumberKey.Int(line))
			}
		}
	}

	if !logger.DisableStacktrace {
		stackTrace := make([]byte, 2048)
		n := runtime.Stack(stackTrace, false)
		attrs = append(attrs, semconv.ExceptionStacktraceKey.String(string(stackTrace[0:n])))
	}

	span.AddEvent("log", trace.WithAttributes(attrs...))

	if lvl >= l.errorStatusLevel {
		span.SetStatus(codes.Error, msg)
	}
}

func runtimeCaller(skip int) (fn, file string, line int, ok bool) {
	rpc := make([]uintptr, 1)
	n := runtime.Callers(skip+1, rpc[:])
	if n < 1 {
		return
	}
	frame, _ := runtime.CallersFrames(rpc).Next()
	return frame.Function, frame.File, frame.Line, frame.PC != 0
}

func appendField(attrs []attribute.KeyValue, f zapcore.Field) []attribute.KeyValue {
	switch f.Type {
	case zapcore.BoolType:
		attr := attribute.Bool(f.Key, f.Integer == 1)
		return append(attrs, attr)

	case zapcore.Int8Type, zapcore.Int16Type, zapcore.Int32Type, zapcore.Int64Type,
		zapcore.Uint32Type, zapcore.Uint8Type, zapcore.Uint16Type, zapcore.Uint64Type,
		zapcore.UintptrType:
		attr := attribute.Int64(f.Key, f.Integer)
		return append(attrs, attr)

	case zapcore.Float32Type, zapcore.Float64Type:
		attr := attribute.Float64(f.Key, math.Float64frombits(uint64(f.Integer)))
		return append(attrs, attr)

	case zapcore.Complex64Type:
		s := strconv.FormatComplex(complex128(f.Interface.(complex64)), 'E', -1, 64)
		attr := attribute.String(f.Key, s)
		return append(attrs, attr)
	case zapcore.Complex128Type:
		s := strconv.FormatComplex(f.Interface.(complex128), 'E', -1, 128)
		attr := attribute.String(f.Key, s)
		return append(attrs, attr)

	case zapcore.StringType:
		attr := attribute.String(f.Key, f.String)
		return append(attrs, attr)
	case zapcore.BinaryType, zapcore.ByteStringType:
		attr := attribute.String(f.Key, string(f.Interface.([]byte)))
		return append(attrs, attr)
	case zapcore.StringerType:
		attr := attribute.String(f.Key, f.Interface.(fmt.Stringer).String())
		return append(attrs, attr)

	case zapcore.DurationType, zapcore.TimeType:
		attr := attribute.Int64(f.Key, f.Integer)
		return append(attrs, attr)
	case zapcore.TimeFullType:
		attr := attribute.Int64(f.Key, f.Interface.(time.Time).UnixNano())
		return append(attrs, attr)
	case zapcore.ErrorType:
		err := f.Interface.(error)
		typ := reflect.TypeOf(err).String()
		attrs = append(attrs, semconv.ExceptionTypeKey.String(typ))
		attrs = append(attrs, semconv.ExceptionMessageKey.String(err.Error()))
		return attrs
	case zapcore.ReflectType:
		attr := opentelemetry.Attribute(f.Key, f.Interface)
		return append(attrs, attr)
	case zapcore.SkipType:
		return attrs

	case zapcore.ArrayMarshalerType:
		var attr attribute.KeyValue
		arrayEncoder := &bufferArrayEncoder{
			stringsSlice: []string{},
		}
		err := f.Interface.(zapcore.ArrayMarshaler).MarshalLogArray(arrayEncoder)
		if err != nil {
			attr = attribute.String(f.Key+"_error", fmt.Sprintf("otelzap: unable to marshal array: %v", err))
		} else {
			attr = attribute.StringSlice(f.Key, arrayEncoder.stringsSlice)
		}
		return append(attrs, attr)

	case zapcore.ObjectMarshalerType:
		attr := attribute.String(f.Key+"_error", "otelzap: zapcore.ObjectMarshalerType is not implemented")
		return append(attrs, attr)

	default:
		attr := attribute.String(f.Key+"_error", fmt.Sprintf("otelzap: unknown field type: %v", f))
		return append(attrs, attr)
	}
}

func levelString(lvl zapcore.Level) string {
	if lvl == zapcore.DPanicLevel {
		return "PANIC"
	}
	return lvl.CapitalString()
}
