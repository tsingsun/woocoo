package web

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
)

// Option the function to apply a configuration option
type Option func(s *ServerOptions)

// WithConfiguration set up the configuration of the web server by a configuration instance
func WithConfiguration(cfg *conf.Configuration) Option {
	return func(s *ServerOptions) {
		s.configuration = cfg
	}
}

// WithMiddlewareNewFunc provide a simple way to inject middleware by MiddlewareNewFunc.
func WithMiddlewareNewFunc(name string, newFunc handler.MiddlewareNewFunc) Option {
	return func(s *ServerOptions) {
		s.handlerManager.Register(name, newFunc)
	}
}

// WithMiddlewareApplyFunc provide a simple way to inject middleware by gin.HandlerFunc.
//
// Notice: the middleware usual attach `c.Next()` or `c.Abort` to indicator whether exits the method.
// example:
//
//		RegisterMiddleware("test", func(cfg *conf.Configuration){
//	     // use cfg to init
//		    return func(c *gin.Context) {
//		        // ....process
//		        c.Next() or c.Abort() or c.AbortWithStatus(500)
//		    }
//		})
func WithMiddlewareApplyFunc(name string, handlerFunc handler.MiddlewareApplyFunc) Option {
	return func(s *ServerOptions) {
		s.handlerManager.Register(name, handler.WrapMiddlewareApplyFunc(name, handlerFunc))
	}
}

// WithGracefulStop indicate use graceful stop
func WithGracefulStop() Option {
	return func(s *ServerOptions) {
		s.gracefulStop = true
	}
}
