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
	// Name returns the name of the handler.
	Name() string
	// ApplyFunc return a gin's handler function by a configuration
	ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc
	// Shutdown the handler,usually call in server quit. some base on file,network may need release the resource
	Shutdown()
}

type Manager struct {
	handlers map[string]Handler
}

func NewManager() *Manager {
	mgr := &Manager{
		handlers: make(map[string]Handler),
	}
	mgr.registerIntegrationHandler()
	return mgr
}

// RegisterHandlerFunc registry a handler middleware
//
// you can override exists handler
func (m *Manager) RegisterHandlerFunc(name string, handler Handler) {
	if _, ok := m.handlers[name]; ok {
		log.Infof("handler override:%s", name)
	}
	m.handlers[name] = handler
}

func (m *Manager) GetHandler(name string) (Handler, bool) {
	h, ok := m.handlers[name]
	return h, ok
}

func integration() map[string]Handler {
	var handlerMap = map[string]Handler{
		recovery.RecoveryHandlerName: recovery.New(),
		auth.AuthHandlerName:         auth.New(),
		logger.LoggerHandlerName:     logger.New(),
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
	for _, handler := range m.handlers {
		handler.Shutdown()
	}
}
