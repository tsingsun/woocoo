package logger

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"path"
	"time"
)

var (
	// SystemField is used in every log statement made through grpc_zap. Can be overwritten before any initialization code.
	SystemField = zap.String("system", "grpc")

	// ServerField is used in every server-side log statement made through grpc_zap.Can be overwritten before initialization.
	ServerField = zap.String("span.kind", "server")
)

type loggerIncomingKey struct{}

func UnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	o := defautlOptions
	o.Apply(cfg)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		newCtx := newLoggerForCall(ctx, info.FullMethod, start, o.TimestampFormat)
		resp, err := handler(newCtx, req)
		ctxloger, ok := FromIncomingContext(newCtx)
		latency := time.Since(start)
		code := status.Code(err)
		level := DefaultCodeToLevel(code)
		fs := []zap.Field{
			zap.Int("status", int(code)),
			zap.Duration("latency", latency),
			zap.Error(err),
		}
		if ok {
			fs = append(fs, ctxloger.fields...)
		}
		log.Operator().Check(level, info.FullMethod).Write(fs...)
		return resp, err
	}
}

type contextLogger struct {
	fields []zapcore.Field
}

func New() (*contextLogger, error) {
	return &contextLogger{}, nil
}

func (c *contextLogger) Apply(cfg *conf.Configuration) {
	if err := cfg.Parser().UnmarshalByJson("", c); err != nil {
		panic(err)
	}
}

func newLoggerForCall(ctx context.Context, fullMethodString string, start time.Time, timestampFormat string) context.Context {
	var f []zapcore.Field
	f = append(f, zap.String("grpc.start_time", start.Format(timestampFormat)))
	if d, ok := ctx.Deadline(); ok {
		f = append(f, zap.String("grpc.request.deadline", d.Format(timestampFormat)))
	}
	if cl, ok := peer.FromContext(ctx); ok {
		f = append(f, zap.Any("peer.address", cl.Addr.String()))
	}
	callLog := &contextLogger{fields: append(f, serverCallFields(fullMethodString)...)}
	return AppendToContext(ctx, callLog)
}

func AppendToContext(ctx context.Context, logger *contextLogger, fields ...zap.Field) context.Context {
	if len(fields) > 0 {
		logger.fields = append(logger.fields, fields...)
	}
	return context.WithValue(ctx, loggerIncomingKey{}, logger)
}

func FromIncomingContext(ctx context.Context) (*contextLogger, bool) {
	fs, ok := ctx.Value(loggerIncomingKey{}).(*contextLogger)
	if !ok {
		return nil, false
	}
	return fs, true
}

func serverCallFields(fullMethodString string) []zapcore.Field {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)
	return []zapcore.Field{
		SystemField,
		ServerField,
		zap.String("grpc.service", service),
		zap.String("grpc.method", method),
	}
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func DefaultCodeToLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zap.InfoLevel
	case codes.Canceled:
		return zap.InfoLevel
	case codes.Unknown:
		return zap.ErrorLevel
	case codes.InvalidArgument:
		return zap.InfoLevel
	case codes.DeadlineExceeded:
		return zap.WarnLevel
	case codes.NotFound:
		return zap.InfoLevel
	case codes.AlreadyExists:
		return zap.InfoLevel
	case codes.PermissionDenied:
		return zap.WarnLevel
	case codes.Unauthenticated:
		return zap.InfoLevel // unauthenticated requests can happen
	case codes.ResourceExhausted:
		return zap.WarnLevel
	case codes.FailedPrecondition:
		return zap.WarnLevel
	case codes.Aborted:
		return zap.WarnLevel
	case codes.OutOfRange:
		return zap.WarnLevel
	case codes.Unimplemented:
		return zap.ErrorLevel
	case codes.Internal:
		return zap.ErrorLevel
	case codes.Unavailable:
		return zap.WarnLevel
	case codes.DataLoss:
		return zap.ErrorLevel
	default:
		return zap.ErrorLevel
	}
}
