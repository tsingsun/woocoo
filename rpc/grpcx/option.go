package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"sync"
)

var once sync.Once

type Option func(s *serverOptions)

func WithConfiguration(cfg *conf.Configuration) Option {
	return func(s *serverOptions) {
		s.configuration = cfg
	}
}

// UseLogger use component logger named "grpc" and set grpclog use zap logger.It is used by default.
func UseLogger() Option {
	return func(s *serverOptions) {
		lg := log.Component(log.GrpcComponentName).Logger()
		if _, ok := lg.ContextLogger().(*log.DefaultContextLogger); ok {
			lg.SetContextLogger(interceptor.NewGrpcContextLogger())
		}
		zgl := zapgrpc.NewLogger(lg.Operator())
		once.Do(func() {
			grpclog.SetLoggerV2(zgl)
		})
	}
}

func WithGrpcOption(opts ...grpc.ServerOption) Option {
	return func(s *serverOptions) {
		s.grpcOptions = append(s.grpcOptions, opts...)
	}
}

// WithGracefulStop indicate use graceful stop
func WithGracefulStop() Option {
	return func(s *serverOptions) {
		s.gracefulStop = true
	}
}
