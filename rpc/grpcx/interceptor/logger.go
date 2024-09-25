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

var (
	defaultLoggerFormat = "grpc.start_time,grpc.service,grpc.method,grpc.request.deadline,status,error,latency," +
		"peer.address,request,response"
	defaultLoggerOptions = LoggerOptions{
		Format: defaultLoggerFormat,
	}
)

type (
	LoggerOptions struct {
		Format string `json:"format" yaml:"format"`
		logger log.ComponentLogger

		tags []string
	}
	AccessLogger struct {
	}
)

// Apply set access log from grpc logger
func (o *LoggerOptions) Apply(cnf *conf.Configuration) {
	if err := cnf.Unmarshal(o); err != nil {
		panic(err)
	}
	lc := log.Component(AccessLogComponentName)
	operator := lc.Logger(log.WithOriginalLogger()).WithOptions(zap.AddStacktrace(zapcore.FatalLevel + 1))
	// reuse grpc logger
	lcl := log.Component(log.GrpcComponentName).Logger()
	operator.SetContextLogger(lcl.ContextLogger())
	lc.SetLogger(operator)
	o.logger = lc
	o.tags = strings.Split(o.Format, ",")
	for i := range o.tags {
		o.tags[i] = strings.TrimSpace(o.tags[i])
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
		newCtx := al.newLoggerForCall(ctx)
		resp, err := handler(newCtx, req)
		latency := time.Since(start)
		fields := make([]zap.Field, len(o.tags))
		code := status.Code(err)
		level := DefaultCodeToLevel(code)
		msg := ""
		for i, tag := range o.tags {
			switch tag {
			case "grpc.start_time":
				fields[i] = zap.Time("grpc.start_time", start)
			case "grpc.request.deadline":
				if d, ok := ctx.Deadline(); ok {
					fields[i] = zap.Time("grpc.request.deadline", d)
				} else {
					fields[i] = zap.Skip()
				}
			case "peer.address":
				if cl, ok := peer.FromContext(ctx); ok {
					fields[i] = zap.Any("peer.address", cl.Addr.String())
				} else {
					fields[i] = zap.Skip()
				}
			case "grpc.service":
				service := path.Dir(info.FullMethod)[1:]
				fields[i] = zap.String("grpc.service", service)
			case "grpc.method":
				method := path.Base(info.FullMethod)
				fields[i] = zap.String("grpc.method", method)
			case "status":
				fields[i] = zap.Int("status", int(code))
			case "error":
				if err != nil {
					fields[i] = zap.Error(err)
					msg = err.Error()
				} else {
					fields[i] = zap.Skip()
				}
			case "latency":
				fields[i] = zap.Duration("latency", latency)
			case "request":
				fields[i] = zap.Any("request", req)
			case "response":
				fields[i] = zap.Any("response", resp)
			default:
				fields[i] = zap.Skip()
			}
		}
		carr, _ := log.FromIncomingContext(newCtx)
		fields = append(fields, carr.Fields...)
		o.logger.Ctx(newCtx).Log(level, msg, fields)
		return resp, err
	}
}

// init context with base fields for log method called.
func (al AccessLogger) newLoggerForCall(ctx context.Context) context.Context {
	callLog := &log.FieldCarrier{}
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

// StreamServerInterceptor returns a new streaming server interceptors that adds a zap.Logger to the context.
func (al AccessLogger) StreamServerInterceptor(cnf *conf.Configuration) grpc.StreamServerInterceptor {
	o := defaultLoggerOptions
	o.Apply(cnf)
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := stream.Context()
		newCtx := al.newLoggerForCall(ctx)
		wrapped := WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		err := handler(srv, wrapped)
		latency := time.Since(start)
		fields := make([]zap.Field, len(o.tags))
		code := status.Code(err)
		level := DefaultCodeToLevel(code)
		msg := ""
		for i, tag := range o.tags {
			switch tag {
			case "grpc.start_time":
				fields[i] = zap.Time("grpc.start_time", start)
			case "grpc.request.deadline":
				if d, ok := ctx.Deadline(); ok {
					fields[i] = zap.Time("grpc.request.deadline", d)
				} else {
					fields[i] = zap.Skip()
				}
			case "peer.address":
				if cl, ok := peer.FromContext(ctx); ok {
					fields[i] = zap.Any("peer.address", cl.Addr.String())
				} else {
					fields[i] = zap.Skip()
				}
			case "grpc.service":
				service := path.Dir(info.FullMethod)[1:]
				fields[i] = zap.String("grpc.service", service)
			case "grpc.method":
				method := path.Base(info.FullMethod)
				fields[i] = zap.String("grpc.method", method)
			case "status":
				fields[i] = zap.Int("status", int(code))
			case "error":
				if err != nil {
					fields[i] = zap.Error(err)
					msg = err.Error()
				} else {
					fields[i] = zap.Skip()
				}
			case "latency":
				fields[i] = zap.Duration("latency", latency)
			default:
				fields[i] = zap.Skip()
			}
		}
		carr, _ := log.FromIncomingContext(newCtx)
		fields = append(fields, carr.Fields...)
		o.logger.Ctx(newCtx).Log(level, msg, fields)
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
