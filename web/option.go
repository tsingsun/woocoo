package web

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
)

// Option the function to apply configuration option
type Option func(s *ServerOptions)

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

// RegisterMiddlewareByFunc provide a simple way to inject a middleware by gin.HandlerFunc.
//
// Notice: the middleware usual attach `c.Next()` or `c.Abort` to indicator whether exits the method.
// example:
//
//	RegisterMiddlewareByFunc("test",func(c *gin.Context) {
//	        ....process
//	        c.Next() or c.Abort() or c.AbortWithStatus(500)
//	    }
//
// )
func RegisterMiddlewareByFunc(name string, handlerFunc handler.MiddlewareApplyFunc) Option {
	ware := handler.NewSimpleMiddleware(name, handlerFunc)
	return RegisterMiddleware(ware)
}

// GracefulStop indicate use graceful stop
func GracefulStop() Option {
	return func(s *ServerOptions) {
		s.gracefulStop = true
	}
}
