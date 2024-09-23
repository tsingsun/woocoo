package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"sync"
)

var once sync.Once

type Option func(s *serverOptions)

func WithConfiguration(cfg *conf.Configuration) Option {
	return func(s *serverOptions) {
		s.configuration = cfg
	}
}

// WithGrpcLogger set grpclog.LoggerV2 with the component logger named "grpc".
// if you need to see grpc log, you should call this option.
//
// notice: call this while go test may cause race condition.
func WithGrpcLogger() Option {
	return func(s *serverOptions) {
		lg := logger.Logger()
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

// WithListener indicate use listener. if set listener, it will ignore the address setting.
func WithListener(lis net.Listener) Option {
	return func(s *serverOptions) {
		s.listener = lis
	}
}
