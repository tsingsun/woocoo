package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/contrib/opentelemetry/otelweb"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler/gql"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

// User demo
type User struct {
	UserName  string
	FirstName string
	LastName  string
}

func main() {
	cfg := conf.New().Load()
	log.NewBuiltIn()
	httpSvr := web.New(web.Configuration(cfg.Sub("web")),
		web.RegisterMiddleware(otelweb.New()),
		web.RegisterMiddleware(gql.New()),
		web.GracefulStop(),
	)
	r := httpSvr.Router().Engine
	r.GET("/", func(c *gin.Context) {
		c.String(200, "hello world")
	})

	r.GET("/abort", func(c *gin.Context) {
		c.Abort()
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})
	if err := httpSvr.Run(); err != nil {
		log.Error(err)
	}
}
