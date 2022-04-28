package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
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

// UseLogger use global logger and set grpc logger with woocoo logger
func UseLogger() Option {
	return func(s *serverOptions) {
		l := log.Global()
		lg := l.With(zap.String("system", "grpc"),
			zap.Bool("grpc_log", true),
		).Operator().WithOptions(zap.AddCallerSkip(2))
		zgl := zapgrpc.NewLogger(lg)
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

// GracefulStop indicate use gracefull stop
func GracefulStop() Option {
	return func(s *serverOptions) {
		s.gracefulStop = true
	}
}
