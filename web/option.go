package web

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

// Option the function to apply configuration option
type Option func(s *Server)

func Config(cnfops ...conf.Option) Option {
	return func(s *Server) {
		s.configuration = conf.New(cnfops...).Load()
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
	}
}
