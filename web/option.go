package web

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
)

// Option the function to apply configuration option
type Option func(s *serverOptions)

// Config set up the configuration of the web server by configuration option.it will be initial Application level Configuration
// and use "web" path for web server.
func Config(cnfops ...conf.Option) Option {
	return func(s *serverOptions) {
		s.configuration = conf.New(cnfops...).Load().Sub("web")
	}
}

// Configuration set up the configuration of the web server by a configuration instance
func Configuration(cfg *conf.Configuration) Option {
	return func(s *serverOptions) {
		s.configuration = cfg
	}
}

// RegisterHandler inject a handler to server,then can be used in Server.Apply method
func RegisterHandler(handler handler.Handler) Option {
	return func(s *serverOptions) {
		s.handlerManager.RegisterHandlerFunc(handler.Name(), handler)
	}
}

// GracefulStop indicate use gracefull stop
func GracefulStop() Option {
	return func(s *serverOptions) {
		s.gracefulStop = true
	}
}
