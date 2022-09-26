package interceptor

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"path"
	"time"
)

var (
	// ServerField is used in every server-side log statement made through grpc_zap.Can be overwritten before initialization.
	ServerField = zap.String("span.kind", "server")

	logger = log.Component("grpc", ServerField)
)

type loggerIncomingKey struct{}

var defaultLoggerOptions = &LoggerOptions{
	TimestampFormat: time.RFC3339,
}

type LoggerOptions struct {
	TimestampFormat string `json:"timestampFormat" yaml:"timestampFormat"`
}

func (o *LoggerOptions) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(o); err != nil {
		panic(err)
	}
}

func LoggerUnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	o := defaultLoggerOptions
	o.Apply(cfg)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		newCtx := newLoggerForCall(ctx, info.FullMethod, start, o.TimestampFormat)
		resp, err := handler(newCtx, req)

		latency := time.Since(start)
		loggerOutPut(newCtx, info.FullMethod, latency, err)
		return resp, err
	}
}

func loggerOutPut(ctx context.Context, method string, latency time.Duration, err error) {
	// must be ok
	carr, _ := carrierFromIncomingContext(ctx)
	code := status.Code(err)
	level := DefaultCodeToLevel(code)
	carr.fields = append(carr.fields,
		zap.Int("status", int(code)),
		zap.Duration("latency", latency),
	)
	if err != nil {
		carr.fields = append(carr.fields, zap.Error(err))
	}
	logger.Ctx(ctx).Log(level, method, carr.fields)
}

func LoggerStreamServerInterceptor(cfg *conf.Configuration) grpc.StreamServerInterceptor {
	o := defaultLoggerOptions
	o.Apply(cfg)
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		newCtx := newLoggerForCall(stream.Context(), info.FullMethod, start, o.TimestampFormat)
		wrapped := WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		err := handler(srv, wrapped)
		latency := time.Since(start)
		loggerOutPut(newCtx, info.FullMethod, latency, err)
		return err
	}
}

// logFieldCarrier sample to carry context log
type logFieldCarrier struct {
	fields []zapcore.Field
}

func (c *logFieldCarrier) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(c); err != nil {
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
	callLog := &logFieldCarrier{fields: append(f, serverCallFields(fullMethodString)...)}
	return AppendLoggerToContext(ctx, callLog)
}

// AppendLoggerFieldToContext appends zap field to context logger
func AppendLoggerFieldToContext(ctx context.Context, fields ...zap.Field) {
	ctxlog, ok := carrierFromIncomingContext(ctx)
	if ok {
		ctxlog.fields = append(ctxlog.fields, fields...)
	}
}

// AppendLoggerToContext initial a logger to context
func AppendLoggerToContext(ctx context.Context, logger *logFieldCarrier, fields ...zap.Field) context.Context {
	if len(fields) > 0 {
		logger.fields = append(logger.fields, fields...)
	}
	return context.WithValue(ctx, loggerIncomingKey{}, logger)
}

// carrierFromIncomingContext returns the logger stored in ctx, if any.
func carrierFromIncomingContext(ctx context.Context) (*logFieldCarrier, bool) {
	fs, ok := ctx.Value(loggerIncomingKey{}).(*logFieldCarrier)
	if !ok {
		return nil, false
	}
	return fs, true
}

func serverCallFields(fullMethodString string) []zapcore.Field {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)
	return []zapcore.Field{
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

type GrpcContextLogger struct {
}

func NewGrpcContextLogger() *GrpcContextLogger {
	return &GrpcContextLogger{}
}

func (g *GrpcContextLogger) LogFields(logger *log.Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) []zap.Field {
	if logger.WithTraceID {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			tid := md.Get(log.TraceID)
			if len(tid) != 0 {
				fields = append(fields, zap.String(log.TraceID, tid[0]))
			}
		}
	}
	return fields
}
