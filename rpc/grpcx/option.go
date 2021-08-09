package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/cache/redis"
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
		var err error
		if s.configuration, err = conf.BuildWithOption(cnfops...); err != nil {
			panic(err)
		}
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
	logger := &log.Logger{}
	return func(s *Server) {
		logger.Apply(s.configuration, "log")
		s.logger = logger
		lg := logger.With(zap.String("system", "grpc"),
			zap.Bool("grpc_log", true),
		).Operator().WithOptions(zap.AddCallerSkip(2))
		zgl := zapgrpc.NewLogger(lg)
		grpclog.SetLoggerV2(zgl)
	}
}

func UseRedisCache() Option {
	rc := &redis.Cache{}
	return Use(rc, "cache")
}

func WithGrpcOption(opts ...grpc.ServerOption) Option {
	return func(s *Server) {
		s.config.grpcOptions = append(s.config.grpcOptions, opts...)
	}
}
