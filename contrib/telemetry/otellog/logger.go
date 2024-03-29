package otellog

import (
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/contrib/telemetry"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math"
	"reflect"
	"time"
)

var (
	logSeverityKey = attribute.Key("log.severity")
	logMessageKey  = attribute.Key("log.message")
)

// ContextLogger is a ContextLogger that add log field to the current recording span.
//
// log configuration set the `callerSkip: 4` for matching the stacktrace
type ContextLogger struct {
	enc *log.TextEncoder
}

func NewContextZapLogger() *ContextLogger {
	return &ContextLogger{
		enc: log.NewTextEncoder(zap.NewProductionEncoderConfig(), true, true, false),
	}
}

func (l *ContextLogger) LogFields(logger *log.Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		logger.Log(lvl, msg, fields...)
		return
	}

	attrs := make([]attribute.KeyValue, len(fields))
	enc := l.enc.Clone().(*log.TextEncoder)
	for _, f := range fields {
		if f.Type == zapcore.NamespaceType {
			continue
		}
		enc.Truncate()
		attrs = appendField(enc, attrs, f)
	}
	ce := l.log(logger, span, lvl, msg, attrs)
	if ce == nil {
		return
	}
	if logger.WithTraceID {
		traceID := span.SpanContext().TraceID().String()
		fields = append(fields, zap.String(logger.TraceIDKey, traceID))
	}
	ce.Write(fields...)
}

func (l *ContextLogger) log(logger *log.Logger, span trace.Span, lvl zapcore.Level, msg string, attrs []attribute.KeyValue) *zapcore.CheckedEntry {
	ce := logger.Operator().Check(lvl, msg)
	if ce == nil {
		return ce
	}
	attrs = append(attrs, logSeverityKey.String(levelString(lvl)), logMessageKey.String(msg))
	if caller := ce.Entry.Caller; caller.Defined {
		attrs = append(attrs,
			semconv.CodeFilepathKey.String(caller.File),
			semconv.CodeFunctionKey.String(caller.Function),
			semconv.CodeLineNumberKey.Int(caller.Line),
		)
	}
	if ce.Entry.Stack != "" {
		attrs = append(attrs, semconv.ExceptionStacktraceKey.String(ce.Entry.Stack))
	}
	span.AddEvent("log", trace.WithAttributes(attrs...))
	if lvl >= zap.ErrorLevel {
		span.SetStatus(codes.Error, msg)
	}
	return ce
}

func appendField(enc *log.TextEncoder, attrs []attribute.KeyValue, f zapcore.Field) []attribute.KeyValue {
	switch f.Type {
	case zapcore.BoolType:
		attr := attribute.Bool(f.Key, f.Integer == 1)
		return append(attrs, attr)
	case zapcore.Int8Type, zapcore.Int16Type, zapcore.Int32Type, zapcore.Int64Type,
		zapcore.Uint32Type, zapcore.Uint8Type, zapcore.Uint16Type, zapcore.Uint64Type,
		zapcore.UintptrType:
		attr := attribute.Int64(f.Key, f.Integer)
		return append(attrs, attr)
	case zapcore.Float32Type:
		attr := attribute.Float64(f.Key, float64(math.Float32frombits(uint32(f.Integer))))
		return append(attrs, attr)
	case zapcore.Float64Type:
		attr := attribute.Float64(f.Key, math.Float64frombits(uint64(f.Integer)))
		return append(attrs, attr)
	case zapcore.Complex64Type:
		enc.AppendComplex64(f.Interface.(complex64))
		attr := attribute.String(f.Key, enc.String())
		return append(attrs, attr)
	case zapcore.Complex128Type:
		enc.AppendComplex128(f.Interface.(complex128))
		attr := attribute.String(f.Key, enc.String())
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
		enc.AppendTime(f.Interface.(time.Time))
		attr := attribute.String(f.Key, enc.String())
		return append(attrs, attr)
	case zapcore.ErrorType:
		err := f.Interface.(error)
		typ := reflect.TypeOf(err).String()
		attrs = append(attrs, semconv.ExceptionTypeKey.String(typ))
		attrs = append(attrs, semconv.ExceptionMessageKey.String(err.Error()))
		return attrs
	case zapcore.ReflectType: // only public field
		attr := telemetry.Attribute(f.Key, f.Interface)
		return append(attrs, attr)
	case zapcore.SkipType:
		return attrs
	case zapcore.ArrayMarshalerType:
		var attr attribute.KeyValue
		err := enc.AppendArray(f.Interface.(zapcore.ArrayMarshaler))
		if err != nil {
			attr = attribute.String(f.Key+"_error", fmt.Sprintf("otelzap: unable to marshal array: %v", err))
		} else {
			attr = attribute.String(f.Key, enc.String())
		}
		return append(attrs, attr)
	case zapcore.ObjectMarshalerType:
		var attr attribute.KeyValue
		err := enc.AppendObject(f.Interface.(zapcore.ObjectMarshaler))
		if err != nil {
			attr = attribute.String(f.Key+"_error", "otelzap: zapcore.ObjectMarshalerType is not implemented")
		} else {
			attr = attribute.String(f.Key, enc.String())
		}
		return append(attrs, attr)
	default:
		var attr attribute.KeyValue
		err := enc.AppendReflected(f.Interface)
		if err != nil {
			attr = attribute.String(f.Key+"_error", fmt.Sprintf("otelzap: unable to encode field: %v", err))
		} else {
			attr = attribute.String(f.Key, enc.String())
		}
		return append(attrs, attr)
	}
}

func levelString(lvl zapcore.Level) string {
	if lvl == zapcore.DPanicLevel {
		return "PANIC"
	}
	return lvl.CapitalString()
}
