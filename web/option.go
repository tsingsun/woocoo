package web

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
)

// Option the function to apply configuration option
type Option func(s *ServerOptions)

// Config set up the configuration of the web server by configuration option.it will be initial Application level Configuration
// and use "web" path for web server.
func Config(cnfops ...conf.Option) Option {
	return func(s *ServerOptions) {
		s.configuration = conf.New(cnfops...).Load().Sub("web")
	}
}

// Configuration set up the configuration of the web server by a configuration instance
func Configuration(cfg *conf.Configuration) Option {
	return func(s *ServerOptions) {
		s.configuration = cfg
	}
}

// RegisterMiddleware inject a handler to server,then can be used in Server.Apply method
func RegisterMiddleware(middleware handler.Middleware) Option {
	return func(s *ServerOptions) {
		s.handlerManager.RegisterHandlerFunc(middleware.Name(), middleware)
	}
}

// GracefulStop indicate use graceful stop
func GracefulStop() Option {
	return func(s *ServerOptions) {
		s.gracefulStop = true
	}
}
