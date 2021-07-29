package main

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/tsingsun/woocoo/third_party/appleboy/gin-jwt"
	"github.com/tsingsun/woocoo/web"
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
	httpSvr := web.Default()
	r := httpSvr.Router().Engine

	r.POST("/login", func(c *gin.Context) {

	})

	if err := httpSvr.Run(true); err != nil {
		log.Fatal(err)
	}

}
