package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
)

//HandlerApplyFunc
type HandlerApplyFunc func(handerCfg *conf.Configuration) gin.HandlerFunc

var RegisterHandler = map[string]HandlerApplyFunc{}
