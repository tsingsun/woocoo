package log

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerIncomingKey struct{}

// FieldCarrier sample to carry context log
type FieldCarrier struct {
	Fields []zapcore.Field
}

func (c *FieldCarrier) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(c); err != nil {
		panic(err)
	}
}

// NewCarrier create a new logger carrier
func NewCarrier() *FieldCarrier {
	return &FieldCarrier{}
}

// AppendLoggerFieldToContext appends zap field to context logger
func AppendLoggerFieldToContext(ctx context.Context, fields ...zap.Field) {
	ctxlog, ok := CarrierFromIncomingContext(ctx)
	if ok {
		ctxlog.Fields = append(ctxlog.Fields, fields...)
	}
}

// WithLoggerCarrierContext initial a logger to context
func WithLoggerCarrierContext(ctx context.Context, logger *FieldCarrier, fields ...zap.Field) context.Context {
	if len(fields) > 0 {
		logger.Fields = append(logger.Fields, fields...)
	}
	return context.WithValue(ctx, loggerIncomingKey{}, logger)
}

// CarrierFromIncomingContext returns the logger stored in ctx, if any.
func CarrierFromIncomingContext(ctx context.Context) (*FieldCarrier, bool) {
	fs, ok := ctx.Value(loggerIncomingKey{}).(*FieldCarrier)
	if !ok {
		return nil, false
	}
	return fs, true
}
