package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
)

func CORS() Middleware {
	return NewSimpleMiddleware(corsName, func(cnf *conf.Configuration) gin.HandlerFunc {
		var config = cors.DefaultConfig()
		config.AllowAllOrigins = true
		if err := cnf.Unmarshal(&config); err != nil {
			panic(err)
		}
		if len(config.AllowOrigins) != 0 || config.AllowOriginFunc != nil {
			config.AllowAllOrigins = false
		}
		return cors.New(config)
	})
}
