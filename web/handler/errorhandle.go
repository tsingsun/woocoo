package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
	"strings"
)

type (
	// ErrorHandleConfig is the config for error handle
	ErrorHandleConfig struct {
		// NegotiateFormat is the offered http media type format for errors.
		// formats split by comma, such as "application/json,application/xml"
		Accepts         string      `json:"accepts" yaml:"negotiateFormat"`
		ErrorParser     ErrorParser `json:"-" yaml:"-"`
		NegotiateFormat []string    `json:"-" yaml:"-"`
	}

	// ErrorParser is the error parser
	ErrorParser func(c *gin.Context) (int, any)
)

var defaultErrorHandleConfig = ErrorHandleConfig{
	NegotiateFormat: []string{binding.MIMEJSON, binding.MIMEXML, binding.MIMEYAML, binding.MIMETOML},
	ErrorParser:     defaultErrorParser,
}

func defaultErrorParser(c *gin.Context) (int, any) {
	var errs = make([]gin.H, len(c.Errors))
	var code = c.Writer.Status()
	for i, e := range c.Errors {
		switch e.Type {
		case gin.ErrorTypePrivate, gin.ErrorTypePublic:
			errs[i] = FormatResponseError(0, e.Err)
			code = 500
		default:
			errs[i] = FormatResponseError(int(e.Type), e.Err)
		}
	}
	return code, gin.H{"errors": errs}
}

// FormatResponseError converts a http error to gin.H
func FormatResponseError(code int, err error) gin.H {
	if code != 0 {
		return gin.H{"code": code, "message": err.Error()}
	}
	return gin.H{"message": err.Error()}
}

// SetContextError set the error to Context,and the error will be handled by ErrorHandleMiddleware
func SetContextError(c *gin.Context, code int, err error) {
	ce := c.Error(err)
	ce.Type = gin.ErrorType(code)
	c.Status(code)
}

// ErrorHandleMiddleware is the middleware for error handle to format the errors to client
type ErrorHandleMiddleware struct {
	config *ErrorHandleConfig
	opts   middlewareOptions
}

// ErrorHandle is the error handle middleware
func ErrorHandle(opts ...MiddlewareOption) *ErrorHandleMiddleware {
	md := &ErrorHandleMiddleware{
		config: new(ErrorHandleConfig),
	}
	md.opts.applyOptions(opts...)
	return md
}

func (em *ErrorHandleMiddleware) Name() string {
	return "errorHandle"
}

func (em *ErrorHandleMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if em.config == nil {
		em.config = new(ErrorHandleConfig)
	}
	*em.config = defaultErrorHandleConfig
	if em.opts.configFunc != nil {
		em.config = em.opts.configFunc().(*ErrorHandleConfig)
	} else if cfg != nil {
		if err := cfg.Unmarshal(&em.config); err != nil {
			panic(err)
		}
	}
	if em.config.Accepts != "" {
		em.config.NegotiateFormat = strings.Split(em.config.Accepts, ",")
	}
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			code, e := em.config.ErrorParser(c)
			NegotiateResponse(c, code, e, em.config.NegotiateFormat)
		}
	}
}

func (em *ErrorHandleMiddleware) Shutdown() {
}

// NegotiateResponse calls different Render according to acceptably Accept format.
//
// no support format described:
//   protobuf: gin.H is not protoreflect.ProtoMessage
func NegotiateResponse(c *gin.Context, code int, data any, offered []string) {
	switch c.NegotiateFormat(offered...) {
	case binding.MIMEJSON:
		c.JSON(code, data)
	case binding.MIMEXML:
		c.XML(code, data)
	case binding.MIMEYAML:
		c.YAML(code, data)
	case binding.MIMETOML:
		c.TOML(code, data)
	default:
		c.AbortWithError(http.StatusNotAcceptable, errors.New("the accepted formats are not offered by the server")) // nolint: errcheck
	}
}
