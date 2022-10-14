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

var defaultLoggerOptions = &LoggerOptions{
	TimestampFormat: time.RFC3339,
}

type LoggerOptions struct {
	TimestampFormat string `json:"timestampFormat" yaml:"timestampFormat"`

	logger log.ComponentLogger
}

func (o *LoggerOptions) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(o); err != nil {
		panic(err)
	}
	operator := logger.Logger().WithOptions(zap.AddStacktrace(zapcore.FatalLevel + 1))
	logger := log.Component(ComponentKey + "." + "accessLog")
	logger.SetLogger(operator)
	o.logger = logger
}

func LoggerUnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	o := defaultLoggerOptions
	o.Apply(cfg)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		newCtx := newLoggerForCall(ctx, info.FullMethod, start, o.TimestampFormat)
		resp, err := handler(newCtx, req)

		latency := time.Since(start)
		loggerOutPut(o.logger, newCtx, info.FullMethod, latency, err)
		return resp, err
	}
}

func loggerOutPut(l log.ComponentLogger, ctx context.Context, method string, latency time.Duration, err error) {
	// must be ok
	carr, _ := log.CarrierFromIncomingContext(ctx)
	code := status.Code(err)
	level := DefaultCodeToLevel(code)
	carr.Fields = append(carr.Fields,
		zap.Int("status", int(code)),
		zap.Duration("latency", latency),
	)
	if err != nil {
		carr.Fields = append(carr.Fields, zap.Error(err))
	}
	clog := l.Ctx(ctx)
	if err != nil {
		if l.Logger().DisableStacktrace && level >= zapcore.ErrorLevel {
			carr.Fields = append(carr.Fields, zap.Stack("stacktrace"))
		}
	}
	clog.Log(level, "", carr.Fields)
	log.PutLoggerWithCtxToPool(clog)
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
		loggerOutPut(o.logger, newCtx, info.FullMethod, latency, err)
		return err
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
	callLog := &log.FieldCarrier{Fields: append(f, serverCallFields(fullMethodString)...)}
	return log.WithLoggerCarrierContext(ctx, callLog)
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
