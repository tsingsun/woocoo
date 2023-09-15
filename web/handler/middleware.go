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

const (
	recoveryName     = "recovery"
	jwtName          = "jwt"
	accessLogName    = "accessLog"
	errorHandlerName = "errorHandle"
	gzipName         = "gzip"
	keyAuthName      = "keyAuth"
	corsName         = "cors"
)

// Middleware is an instance to build middleware for web application.
type Middleware interface {
	// Name returns the name of the handler.
	Name() string
	// ApplyFunc return a gin's handler function by a configuration
	ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc
}

type Shutdown interface {
	// Shutdown the handler, usually call in server quit. Some based on file,network may need to release the resource
	Shutdown(ctx context.Context) error
}

// MiddlewareApplyFunc defines a function to initial new middleware by a configuration
type MiddlewareApplyFunc func(cfg *conf.Configuration) gin.HandlerFunc

// MiddlewareNewFunc defines a function to initial new middleware
type MiddlewareNewFunc func() Middleware

// SimpleMiddleware is a convenience to build middleware by name and gin.HandlerFunc
type SimpleMiddleware struct {
	name      string
	applyFunc MiddlewareApplyFunc
}

// WrapMiddlewareApplyFunc wraps a MiddlewareApplyFunc to MiddlewareNewFunc
func WrapMiddlewareApplyFunc(name string, applyFunc MiddlewareApplyFunc) MiddlewareNewFunc {
	return func() Middleware {
		return NewSimpleMiddleware(name, applyFunc)
	}
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

// Skipper defines a function to skip middleware. Returning true skips processing
// the middleware.
type Skipper func(c *gin.Context) bool

// DefaultSkipper returns false which processes the middleware.
func DefaultSkipper(c *gin.Context) bool {
	return false
}

// PathSkip returns a skipper function that skips middleware if the request path
func PathSkip(list []string, url *url.URL) bool {
	src := strings.TrimRight(url.Path, "/")
	if src == "" || src[0] != '/' {
		src = "/" + src
	}
	for _, skip := range list {
		if skip == src {
			return true
		}
	}
	return false
}

func Gzip() Middleware {
	mw := gzip.NewGzip()
	return mw
}

// Manager is a manager about middleware new function and shutdown function,
// and auto calls those functions when needed; it tries to keep the middleware are immutable.
//
// If you want to get middleware information in your application,
// and you know that your middleware is immutable, or you can control it, you can store it in the cache for reuse.
type Manager struct {
	newFuncs    map[string]MiddlewareNewFunc
	middlewares map[string]Middleware
}

// NewManager creates a new middleware manager, initialize common useful middlewares.
func NewManager() *Manager {
	mgr := &Manager{
		newFuncs:    make(map[string]MiddlewareNewFunc),
		middlewares: make(map[string]Middleware),
	}
	mgr.registerIntegrationHandler()
	return mgr
}

// Register a middleware new function.
func (m *Manager) Register(name string, handler MiddlewareNewFunc) {
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
func (m *Manager) RegisterMiddleware(key string, mid Middleware) {
	if _, ok := m.middlewares[key]; ok {
		panic("middleware could not override")
	}
	m.middlewares[key] = mid
}

func (m *Manager) Get(name string) (MiddlewareNewFunc, bool) {
	v, ok := m.newFuncs[name]
	return v, ok
}

// GetMiddleware returns a middleware instance by key. Should not change the middleware's option value and keep
// middleware run immutable.
func (m *Manager) GetMiddleware(key string) (Middleware, bool) {
	v, ok := m.middlewares[key]
	return v, ok
}

func integration() map[string]MiddlewareNewFunc {
	var handlerMap = map[string]MiddlewareNewFunc{
		recoveryName:     Recovery,
		jwtName:          JWT,
		accessLogName:    AccessLog,
		errorHandlerName: ErrorHandle,
		gzipName:         Gzip,
		keyAuthName:      KeyAuth,
		corsName:         CORS,
	}
	return handlerMap
}

func (m *Manager) registerIntegrationHandler() {
	for s, applyFunc := range integration() {
		m.Register(s, applyFunc)
	}
}

// Shutdown a handler if handler base on file,net such as a need to release resource
func (m *Manager) Shutdown(ctx context.Context) error {
	for key, mid := range m.middlewares {
		if sd, ok := mid.(Shutdown); ok {
			sd.Shutdown(ctx) //nolint:errcheck
		}
		delete(m.middlewares, key)
	}
	return nil
}

// gin.Context is not context.Context, set the store value target
const derivativeContextKey = "woocoo_web_derivative_context"

// SetDerivativeContext set the derivative context to gin.Context
func SetDerivativeContext(c *gin.Context, ctx context.Context) {
	c.Set(derivativeContextKey, ctx)
}

// GetDerivativeContext get the derivative context from gin.Context, return gin.Context if no existing
func GetDerivativeContext(c *gin.Context) context.Context {
	ctx, ok := c.Get(derivativeContextKey)
	if !ok {
		return c
	}
	return ctx.(context.Context)
}

// DerivativeContextWithValue try to set a ctx to the derivative context slot in gin.Context,
// if no existing set to request.Context and return false.
func DerivativeContextWithValue(c *gin.Context, key, val any) bool {
	ctx, ok := c.Get(derivativeContextKey)
	if !ok {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), key, val))
	} else {
		SetDerivativeContext(c, context.WithValue(ctx.(context.Context), key, val))
	}
	return ok
}
