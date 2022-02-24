package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler/auth"
	"github.com/tsingsun/woocoo/web/handler/logger"
	"github.com/tsingsun/woocoo/web/handler/recovery"
)

// Handler is a instance to build a gin middleware for web application.
type Handler interface {
	// ApplyFunc return a gin's handler function by a configuration
	ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc
	// Shutdown the handler,usually call in server quit. some base on file,network may need release the resource
	Shutdown()
}

var RegisterHandler = map[string]Handler{}

// RegisterHandlerFunc registry a handler middleware
//
// you can override exists handler
func RegisterHandlerFunc(name string, handler Handler) {
	if _, ok := RegisterHandler[name]; ok {
		log.Infof("handler override:%s", name)
	}
	RegisterHandler[name] = handler
}

func Integration() map[string]Handler {
	var handlerMap = map[string]Handler{
		"recovery":  recovery.New(),
		"auth":      auth.New(),
		"accessLog": logger.New(),
	}
	return handlerMap
}

func RegisterIntegrationHandler() {
	for s, applyFunc := range Integration() {
		RegisterHandlerFunc(s, applyFunc)
	}
}

// Shutdown a handler if handler base on file,net such as need to release resource
func Shutdown() {
	for _, handler := range RegisterHandler {
		handler.Shutdown()
	}
}
