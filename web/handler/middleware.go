package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"net/url"
	"strings"
)

var (
	logger = log.Component(log.WebComponentName)
)

const (
	RecoverName      = "recovery"
	JWTName          = "jwt"
	AccessLogName    = "accessLog"
	ErrorHandlerName = "errorHandle"
	GZipName         = "gzip"
	KeyAuthName      = "keyAuth"
	CORSName         = "cors"
	CSRFName         = "csrf"
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
func PathSkip(pm map[string]struct{}, url *url.URL) bool {
	src := strings.TrimRight(url.Path, "/")
	if src == "" || src[0] != '/' {
		src = "/" + src
	}
	_, ok := pm[src]
	return ok
}

// StringsToMap convert a string slice to a map
func StringsToMap(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, v := range list {
		m[v] = struct{}{}
	}
	return m
}

// PathSkipper returns a Skipper function that skips processing middleware
func PathSkipper(exclude []string) Skipper {
	if len(exclude) == 0 {
		return DefaultSkipper
	}
	pm := StringsToMap(exclude)
	return func(c *gin.Context) bool {
		return PathSkip(pm, c.Request.URL)
	}
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
