package web

import (
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/tsingsun/woocoo/web/handler/gzip"
)

// HandlerManager is a manager about middleware new function and shutdown function,
// and auto calls those functions when needed; it tries to keep the middleware are immutable.
//
// If you want to get middleware information in your application,
// and you know that your middleware is immutable, or you can control it, you can store it in the cache for reuse.
type HandlerManager struct {
	newFuncs    map[string]handler.MiddlewareNewFunc
	middlewares map[string]handler.Middleware
}

// NewHandlerManager creates a new middleware manager, initialize common useful middlewares.
func NewHandlerManager() *HandlerManager {
	mgr := &HandlerManager{
		newFuncs:    make(map[string]handler.MiddlewareNewFunc),
		middlewares: make(map[string]handler.Middleware),
	}
	mgr.registerIntegrationHandler()
	return mgr
}

// Register a middleware new function.
func (m *HandlerManager) Register(name string, handler handler.MiddlewareNewFunc) {
	if _, ok := m.newFuncs[name]; ok {
		logger.Warn(fmt.Sprintf("middlware new func override:%s", name))
	}
	m.newFuncs[name] = handler
}

// GetMiddlewareKey returns a unique key for middleware
func GetMiddlewareKey(group, name string) string {
	if group == "/" {
		group = "default"
	}
	return group + ":" + name
}

// RegisterMiddleware register a middleware instance. Should call it after Register. Keep the key unique.
func (m *HandlerManager) RegisterMiddleware(key string, mid handler.Middleware) {
	if _, ok := m.middlewares[key]; ok {
		panic("middleware could not override")
	}
	m.middlewares[key] = mid
}

func (m *HandlerManager) Get(name string) (handler.MiddlewareNewFunc, bool) {
	v, ok := m.newFuncs[name]
	return v, ok
}

// GetMiddleware returns a middleware instance by key. Should not change the middleware's option value and keep
// middleware run immutable.
func (m *HandlerManager) GetMiddleware(key string) (handler.Middleware, bool) {
	v, ok := m.middlewares[key]
	return v, ok
}

func integration() map[string]handler.MiddlewareNewFunc {
	var handlerMap = map[string]handler.MiddlewareNewFunc{
		handler.RecoverName:      handler.Recovery,
		handler.JWTName:          handler.JWT,
		handler.AccessLogName:    handler.AccessLog,
		handler.ErrorHandlerName: handler.ErrorHandle,
		handler.GZipName:         gzip.Gzip,
		handler.KeyAuthName:      handler.KeyAuth,
		handler.CORSName:         handler.CORS,
	}
	return handlerMap
}

func (m *HandlerManager) registerIntegrationHandler() {
	for s, applyFunc := range integration() {
		m.Register(s, applyFunc)
	}
}

// Shutdown a handler if handler base on file,net such as a need to release resource
func (m *HandlerManager) Shutdown(ctx context.Context) error {
	for key, mid := range m.middlewares {
		if sd, ok := mid.(handler.Shutdown); ok {
			sd.Shutdown(ctx) //nolint:errcheck
		}
		delete(m.middlewares, key)
	}
	return nil
}
