package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type Option func(s *Server)

func Config(cnfops ...conf.Option) Option {
	return func(s *Server) {
		s.configuration = conf.New(cnfops...).Load()
	}
}

func Configuration(configuration *conf.Configuration, configurationKey string) Option {
	return func(s *Server) {
		s.configuration = configuration
		s.configurationKey = configurationKey
	}
}

func Use(configurable conf.Configurable, path string) Option {
	return func(s *Server) {
		configurable.Apply(s.configuration, path)
	}
}

func UseLogger() Option {
	return func(s *Server) {
		s.logger = log.NewBuiltIn()
		lg := s.logger.With(zap.String("system", "grpc"),
			zap.Bool("grpc_log", true),
		).Operator().WithOptions(zap.AddCallerSkip(2))
		zgl := zapgrpc.NewLogger(lg)
		grpclog.SetLoggerV2(zgl)
	}
}

func WithGrpcOption(opts ...grpc.ServerOption) Option {
	return func(s *Server) {
		s.config.grpcOptions = append(s.config.grpcOptions, opts...)
	}
}
