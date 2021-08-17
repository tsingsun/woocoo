package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
)

//HandlerApplyFunc
type HandlerApplyFunc func(handerCfg *conf.Configuration) gin.HandlerFunc

var RegisterHandler = map[string]HandlerApplyFunc{}

func RegisterHandlerFunc(name string, handlerFunc HandlerApplyFunc) {
	//if _, ok := registerHandler[name]; ok {
	//	panic("handlerFunc exists:" + name)
	//}
	RegisterHandler[name] = handlerFunc
}
