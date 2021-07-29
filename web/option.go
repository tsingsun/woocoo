package web

import (
	"github.com/tsingsun/woocoo/pkg/cache/redis"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

// Option the function to apply configuration option
type Option func(s *Server)

func Config(cnfops ...conf.Option) Option {
	return func(s *Server) {
		var err error
		if s.configuration, err = conf.BuildWithOption(cnfops...); err != nil {
			panic(err)
		}
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
	}
}

func UseRedisCache() Option {
	rc := &redis.Cache{}
	return Use(rc, "cache")
}
