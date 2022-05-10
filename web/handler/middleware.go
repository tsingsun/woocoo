package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

var (
	logger = log.Component(log.WebComponentName)
)

// Middleware is an instance to build a echo middleware for web application.
type Middleware interface {
	// Name returns the name of the handler.
	Name() string
	// ApplyFunc return a gin's handler function by a configuration
	ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc
	// Shutdown the handler,usually call in server quit. some base on file,network may need release the resource
	Shutdown()
}

type Manager struct {
	middlewares map[string]Middleware
}

func NewManager() *Manager {
	mgr := &Manager{
		middlewares: make(map[string]Middleware),
	}
	mgr.registerIntegrationHandler()
	return mgr
}

// RegisterHandlerFunc registry a handler middleware
//
// you can override exists handler
func (m *Manager) RegisterHandlerFunc(name string, handler Middleware) {
	if _, ok := m.middlewares[name]; ok {
		log.Infof("handler override:%s", name)
	}
	m.middlewares[name] = handler
}

func (m *Manager) Get(name string) (Middleware, bool) {
	h, ok := m.middlewares[name]
	return h, ok
}

func integration() map[string]Middleware {
	jwt := JWT()
	reco := Recovery()
	acclog := Logger()
	var handlerMap = map[string]Middleware{
		reco.Name():   reco,
		jwt.Name():    jwt,
		acclog.Name(): acclog,
	}
	return handlerMap
}

func (m *Manager) registerIntegrationHandler() {
	for s, applyFunc := range integration() {
		m.RegisterHandlerFunc(s, applyFunc)
	}
}

// Shutdown a handler if handler base on file,net such as need to release resource
func (m *Manager) Shutdown() {
	for _, handler := range m.middlewares {
		handler.Shutdown()
	}
}
