package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerIncomingKey struct{}

// FieldCarrier sample to carry context log, the carrier's fields will log by demand.
type FieldCarrier struct {
	Fields []zapcore.Field
}

// NewCarrier create a new logger carrier
func NewCarrier() *FieldCarrier {
	return &FieldCarrier{}
}

// AppendToIncomingContext appends zap field to context logger
func AppendToIncomingContext(ctx context.Context, fields ...zap.Field) {
	ctxlog, ok := FromIncomingContext(ctx)
	if ok {
		ctxlog.Fields = append(ctxlog.Fields, fields...)
	}
}

// NewIncomingContext creates a new context with logger carrier.
func NewIncomingContext(ctx context.Context, carrier *FieldCarrier, fields ...zap.Field) context.Context {
	if len(fields) > 0 {
		carrier.Fields = append(carrier.Fields, fields...)
	}
	return context.WithValue(ctx, loggerIncomingKey{}, carrier)
}

// FromIncomingContext returns the logger stored in ctx, if any.
func FromIncomingContext(ctx context.Context) (*FieldCarrier, bool) {
	fs, ok := ctx.Value(loggerIncomingKey{}).(*FieldCarrier)
	if !ok {
		return nil, false
	}
	return fs, true
}
