package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"time"
)

// CORS Cross-Origin Resource Sharing (CORS) support
func CORS() Middleware {
	return NewSimpleMiddleware(corsName, func(cnf *conf.Configuration) gin.HandlerFunc {
		config := cors.Config{
			AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type"},
			AllowAllOrigins: true,
			MaxAge:          12 * time.Hour,
		}
		// unmarshal can't clear the slice, set those manually
		if err := cnf.Unmarshal(&config); err != nil {
			panic(err)
		}
		if cnf.IsSet("allowHeaders") {
			config.AllowHeaders = cnf.StringSlice("allowHeaders")
		}
		if cnf.IsSet("allowMethods") {
			config.AllowMethods = cnf.StringSlice("allowMethods")
		}

		if len(config.AllowOrigins) != 0 || config.AllowOriginFunc != nil {
			config.AllowAllOrigins = false
		}
		return cors.New(config)
	})
}
