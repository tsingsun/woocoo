package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
)

type CustomizeRoute struct{}

func (r *CustomizeRoute) InitRule() {
	RegisterRouteRule("/", http.MethodGet, func(context *gin.Context) {})
}

func (r CustomizeRoute) Get() func(context2 *gin.Context) {
	return func(context2 *gin.Context) {
	}
}

func TestRouter_Collect(t *testing.T) {
	r := NewRouter(&serverOptions{})
	r.Engine = gin.New()
	r.Engine.GET("/", func(context *gin.Context) {
		context.JSON(500, nil)
	})
	c := r.Collect()
	if len(c) != 1 {
		t.Error("not collect rule")
	}
}

func TestRouter_AddRule(t *testing.T) {
	r := NewRouter(&serverOptions{})
	r.Engine = gin.New()
	var newfunc gin.HandlerFunc = func(context *gin.Context) {
		context.JSON(500, nil)
	}
	r.Engine.GET("/", newfunc)
	r.Engine.GET("/a", CustomizeRoute{}.Get())
	cr := &CustomizeRoute{}
	cr.InitRule()
	c := r.Collect()
	if len(c) != 2 {
		t.Error("add rule failure!")
	}
}
