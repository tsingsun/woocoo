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
	"strings"
	"time"
)

var defaultLoggerOptions = &LoggerOptions{
	TimestampFormat: time.RFC3339,
}

type (
	LoggerOptions struct {
		TimestampFormat string `json:"timestampFormat" yaml:"timestampFormat"`
		Format          string `json:"format" yaml:"format"`
		logger          log.ComponentLogger

		logRequest  bool
		logResponse bool
	}
	AccessLogger struct {
	}
)

// Apply set access log from grpc logger
func (o *LoggerOptions) Apply(cnf *conf.Configuration) {
	if err := cnf.Unmarshal(o); err != nil {
		panic(err)
	}
	logger := log.Component(AccessLogComponentName)
	operator := logger.Logger(log.WithOriginalLogger()).WithOptions(zap.AddStacktrace(zapcore.FatalLevel + 1))
	cl := log.Component(log.GrpcComponentName).Logger()
	operator.SetContextLogger(cl.ContextLogger())
	logger.SetLogger(operator)
	o.logger = logger
	ts := strings.Split(o.Format, ",")
	for _, tag := range ts {
		switch tag {
		case "request":
			o.logRequest = true
		case "response":
			o.logResponse = true
		}
	}
}

func (AccessLogger) Name() string {
	return "accessLog"
}

// UnaryServerInterceptor returns a new unary server interceptors that adds a zap.Logger to the context.
func (al AccessLogger) UnaryServerInterceptor(cnf *conf.Configuration) grpc.UnaryServerInterceptor {
	o := defaultLoggerOptions
	o.Apply(cnf)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		newCtx := al.newLoggerForCall(ctx, info.FullMethod, start, o.TimestampFormat)
		resp, err := handler(newCtx, req)
		latency := time.Since(start)
		al.finalCollectFields(newCtx, o, latency, err, req, resp)
		return resp, err
	}
}

// init context with base fields for log method called.
func (al AccessLogger) newLoggerForCall(ctx context.Context, fullMethodString string, start time.Time, timestampFormat string) context.Context {
	var f = make([]zapcore.Field, 0, 5)
	f = append(f, zap.String("grpc.start_time", start.Format(timestampFormat)))
	if d, ok := ctx.Deadline(); ok {
		f = append(f, zap.String("grpc.request.deadline", d.Format(timestampFormat)))
	}
	if cl, ok := peer.FromContext(ctx); ok {
		f = append(f, zap.Any("peer.address", cl.Addr.String()))
	}
	callLog := &log.FieldCarrier{Fields: append(f, al.serverCallFields(fullMethodString)...)}
	return log.NewIncomingContext(ctx, callLog)
}

func (al AccessLogger) serverCallFields(fullMethodString string) []zapcore.Field {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)
	return []zapcore.Field{
		zap.String("grpc.service", service),
		zap.String("grpc.method", method),
	}
}

func (al AccessLogger) finalCollectFields(ctx context.Context, opts *LoggerOptions, latency time.Duration,
	err error, req, resp any) {
	code := status.Code(err)
	fds := make([]zap.Field, 0, 10)
	fds = append(fds, zap.Int("status", int(code)), zap.Duration("latency", latency))
	if opts.logRequest {
		fds = append(fds, zap.Any("request", req))
	}
	if opts.logResponse {
		fds = append(fds, zap.Any("response", resp))
	}
	if err != nil {
		fds = append(fds, zap.Error(err))
	}
	level := DefaultCodeToLevel(code)
	// must be ok
	carr, _ := log.FromIncomingContext(ctx)
	fds = append(fds, carr.Fields...)
	opts.logger.Ctx(ctx).Log(level, "", fds)
}

// StreamServerInterceptor returns a new streaming server interceptors that adds a zap.Logger to the context.
func (al AccessLogger) StreamServerInterceptor(cnf *conf.Configuration) grpc.StreamServerInterceptor {
	o := defaultLoggerOptions
	o.Apply(cnf)
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		newCtx := al.newLoggerForCall(stream.Context(), info.FullMethod, start, o.TimestampFormat)
		wrapped := WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		err := handler(srv, wrapped)
		latency := time.Since(start)
		al.finalCollectFields(newCtx, o, latency, err, nil, nil)
		return err
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

// UseContextLogger replace the default context logger with grpc context logger.
func UseContextLogger() *log.Logger {
	glog := log.Component(log.GrpcComponentName).Logger(log.WithContextLogger())
	if glog != nil {
		if _, ok := glog.ContextLogger().(*log.DefaultContextLogger); ok {
			glog.SetContextLogger(NewGrpcContextLogger())
		}
	}
	return glog
}

type GrpcContextLogger struct {
}

func NewGrpcContextLogger() *GrpcContextLogger {
	return &GrpcContextLogger{}
}

func (g *GrpcContextLogger) LogFields(logger *log.Logger, ctx context.Context, lvl zapcore.Level, msg string, fields []zap.Field) {
	if logger.WithTraceID {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			tid := md.Get(log.TraceIDKey)
			if len(tid) != 0 {
				fields = append(fields, zap.String(logger.TraceIDKey, tid[0]))
			}
		}
	}
	logger.Log(lvl, msg, fields...)
}
