package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/contrib/opentelemetry/otelweb"
	"github.com/tsingsun/woocoo/web"
	jwt "github.com/tsingsun/woocoo/web/handler/auth"
	"log"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

var identityKey = "id"

func helloHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	user, _ := c.Get(identityKey)
	c.JSON(200, gin.H{
		"userID":   claims[identityKey],
		"userName": user.(*User).UserName,
		"text":     "Hello World.",
	})
}

// User demo
type User struct {
	UserName  string
	FirstName string
	LastName  string
}

func main() {
	httpSvr := web.NewBuiltIn(web.RegisterHandler("otel", otelweb.New()))
	r := httpSvr.Router().Engine

	r.GET("/", func(c *gin.Context) {
		c.String(200, "hello world")
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	if err := httpSvr.Run(false); err != nil {
		log.Fatal(err)
	}

}
