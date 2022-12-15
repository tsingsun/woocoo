package interceptor

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	AccessLogComponentName = log.GrpcComponentName + ".accessLog"
)

var (
	// ServerField is used in every server-side log statement made through grpc_zap.Can be overwritten before initialization.
	ServerField = zap.String("span.kind", "server")

	logger = log.Component(log.GrpcComponentName, ServerField)
)

type Interceptor interface {
	Name() string
	UnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor
	StreamServerInterceptor(cfg *conf.Configuration) grpc.StreamServerInterceptor
	UnaryClientInterceptor(cfg *conf.Configuration) grpc.UnaryClientInterceptor
	StreamClientInterceptor(cfg *conf.Configuration) grpc.StreamClientInterceptor
	Shutdown(ctx context.Context) error
}

// WrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type WrappedServerStream struct {
	grpc.ServerStream
	// WrappedContext is the wrapper's own Context. You can assign it.
	WrappedContext context.Context
}

// Context returns the wrapper's WrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *WrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// WrapServerStream returns a ServerStream that has the ability to overwrite context.
func WrapServerStream(stream grpc.ServerStream) *WrappedServerStream {
	if existing, ok := stream.(*WrappedServerStream); ok {
		return existing
	}
	return &WrappedServerStream{ServerStream: stream, WrappedContext: stream.Context()}
}
