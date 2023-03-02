package handler

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler/gzip"
	"net/url"
	"strings"
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
	Shutdown(ctx context.Context) error
}

type MiddlewareApplyFunc func(cfg *conf.Configuration) gin.HandlerFunc

// SimpleMiddleware is a convenience to build middleware by name and gin.HandlerFunc
type SimpleMiddleware struct {
	name      string
	applyFunc MiddlewareApplyFunc
}

// NewSimpleMiddleware returns a new SimpleMiddleware instance.
//
// SimpleMiddleware shutdowns method is empty.
// cfg: the configuration of the middleware, usually pass by web server.
func NewSimpleMiddleware(name string, applyFunc MiddlewareApplyFunc) *SimpleMiddleware {
	middleware := &SimpleMiddleware{
		name:      name,
		applyFunc: applyFunc,
	}
	return middleware
}

func (s *SimpleMiddleware) Name() string {
	return s.name
}

func (s *SimpleMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	return s.applyFunc(cfg)
}

func (s *SimpleMiddleware) Shutdown(_ context.Context) error {
	return nil
}

// Skipper defines a function to skip middleware. Returning true skips processing
// the middleware.
type Skipper func(c *gin.Context) bool

// DefaultSkipper returns false which processes the middleware.
func DefaultSkipper(c *gin.Context) bool {
	return false
}

// PathSkip returns a skipper function that skips middleware if the request path
func PathSkip(list []string, url *url.URL) bool {
	src := url.Path
	if src != "/" && strings.HasSuffix(src, "/") {
		src = url.Path[:len(url.Path)-1]
	}
	for _, skip := range list {
		if skip == src {
			return true
		}
	}
	return false
}

// Manager is a middleware manager
type Manager struct {
	middlewares map[string]Middleware
}

// NewManager creates a new middleware manager, initialize common useful middlewares.
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
		logger.Warn(fmt.Sprintf("handler override:%s", name))
	}
	m.middlewares[name] = handler
}

// Get returns a handler middleware by name
func (m *Manager) Get(name string) (Middleware, bool) {
	h, ok := m.middlewares[name]
	return h, ok
}

func integration() map[string]Middleware {
	jwt := JWT()
	reco := Recovery()
	acclog := AccessLog()
	errhandle := ErrorHandle()
	gz := gzip.Gzip()
	ka := KeyAuth()
	var handlerMap = map[string]Middleware{
		reco.Name():      reco,
		jwt.Name():       jwt,
		acclog.Name():    acclog,
		errhandle.Name(): errhandle,
		gz.Name():        gz,
		ka.Name():        ka,
	}
	return handlerMap
}

func (m *Manager) registerIntegrationHandler() {
	for s, applyFunc := range integration() {
		m.RegisterHandlerFunc(s, applyFunc)
	}
}

// Shutdown a handler if handler base on file,net such as need to release resource
func (m *Manager) Shutdown(ctx context.Context) error {
	for _, handler := range m.middlewares {
		handler.Shutdown(ctx) // lint:ignore
	}
	return nil
}
