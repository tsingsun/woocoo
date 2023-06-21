package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/tsingsun/woocoo/pkg/conf"
)

var (
	DefaultNegotiateFormat = []string{binding.MIMEJSON, binding.MIMEXML}
)

type (
	// ErrorHandleConfig is the config for error handle
	ErrorHandleConfig struct {
		// NegotiateFormat is the offered http media type format for errors.
		// formats split by comma, such as "application/json,application/xml"
		Accepts         string      `json:"accepts" yaml:"accepts"`
		NegotiateFormat []string    `json:"-" yaml:"-"`
		ErrorParser     ErrorParser `json:"-" yaml:"-"`
		// Message is the string while the error is private
		Message string `json:"message" yaml:"message"`
	}

	// ErrorParser is the error parser,public error adopt by private error to show to client.
	ErrorParser func(c *gin.Context, public error) (int, any)

	// ErrorHandleMiddleware is the middleware for error handle to format the errors to client
	ErrorHandleMiddleware struct {
		config *ErrorHandleConfig
		opts   middlewareOptions
	}
)

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
	if em.opts.configFunc != nil {
		em.config = em.opts.configFunc().(*ErrorHandleConfig)
	} else if cfg != nil {
		if err := cfg.Unmarshal(&em.config); err != nil {
			panic(err)
		}
	}
	if em.config.Accepts != "" {
		em.config.NegotiateFormat = strings.Split(em.config.Accepts, ",")
	} else if len(em.config.NegotiateFormat) == 0 {
		em.config.NegotiateFormat = DefaultNegotiateFormat
	}
	var messageError error
	if em.config.Message != "" {
		messageError = errors.New(em.config.Message)
	}
	if em.config.ErrorParser == nil {
		em.config.ErrorParser = DefaultErrorParser
	}
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			code, e := em.config.ErrorParser(c, messageError)
			NegotiateResponse(c, code, e, em.config.NegotiateFormat)
		}
	}
}

var DefaultErrorParser ErrorParser = func(c *gin.Context, public error) (int, any) {
	var errs = make([]gin.H, len(c.Errors))
	var code = c.Writer.Status()
	if code == http.StatusOK {
		code = http.StatusInternalServerError
	}
	for i, e := range c.Errors {
		switch e.Type {
		case gin.ErrorTypePublic:
			errs[i] = FormatResponseError(code, e.Err)
		case gin.ErrorTypePrivate:
			if public == nil {
				errs[i] = FormatResponseError(code, e.Err)
			} else {
				errs[i] = FormatResponseError(code, public)
			}
		default:
			errs[i] = FormatResponseError(int(e.Type), e.Err)
		}
	}
	return code, gin.H{"errors": errs}
}

func (em *ErrorHandleMiddleware) Shutdown(_ context.Context) error {
	return nil
}

// NegotiateResponse calls different Render according to acceptably Accept format.
//
// HTTP Status Code behavior: 1. response has sent can not change it.  2. if code not default,use it.
// 3. if code is default,use passed code.
//
// no support format described:
//
//	protobuf: gin.H is not protoreflect.ProtoMessage
func NegotiateResponse(c *gin.Context, code int, data any, offered []string) {
	if c.Writer.Written() { // if the status has been written,must not write again
		code = c.Writer.Status()
	} else if c.Writer.Status() != http.StatusOK {
		code = c.Writer.Status()
	}
	switch c.NegotiateFormat(offered...) {
	case binding.MIMEJSON:
		c.JSON(code, data)
	case binding.MIMEXML:
		c.XML(code, data)
	default:
		if c.Writer.Written() {
			c.Error(errors.New("the accepted formats are not offered by the server")) // nolint: errcheck
		} else {
			c.AbortWithError(http.StatusNotAcceptable, errors.New("the accepted formats are not offered by the server")) // nolint: errcheck
		}
	}
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
}

// AbortWithError gin context's AbortWithError prevent to reset response header, so we need to replace it
func AbortWithError(c *gin.Context, code int, err error) {
	c.Abort()
	c.Error(err) // nolint: errcheck
	c.Status(code)
}
