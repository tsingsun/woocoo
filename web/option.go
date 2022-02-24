package web

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

// Option the function to apply configuration option
type Option func(s *Server)

// Config set up the configuration of the web server by configuration option.
func Config(cnfops ...conf.Option) Option {
	return func(s *Server) {
		s.configuration = conf.New(cnfops...).Load()
	}
}

// Configuration set up the configuration of the web server by a configuration instance
func Configuration(cfg *conf.Configuration) Option {
	return func(s *Server) {
		s.configuration = cfg
	}
}

func Use(configurable conf.Configurable, path string) Option {
	return func(s *Server) {
		configurable.Apply(s.configuration, path)
	}
}

// UseLogger indicate the web server using a builtin logger which setup by config file
func UseLogger() Option {
	return func(s *Server) {
		s.logger = log.NewBuiltIn()
	}
}
