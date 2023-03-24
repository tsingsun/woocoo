package handler

import (
	"context"
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
		NegotiateFormat []string    `json:"-" yaml:"-"`
		ErrorParser     ErrorParser `json:"-" yaml:"-"`
		// Message is the string while the error is private
		Message      string `json:"message" yaml:"message"`
		messageError error
	}

	// ErrorParser is the error parser
	ErrorParser func(c *gin.Context, public error) (int, any)
)

var DefaultErrorHandleConfig = ErrorHandleConfig{
	NegotiateFormat: []string{binding.MIMEJSON, binding.MIMEXML, binding.MIMEHTML},
	ErrorParser:     defaultErrorParser,
}

// FormatResponseError converts a http error to gin.H
func FormatResponseError(code int, err error) gin.H {
	if code != 0 {
		return gin.H{"code": code, "message": err.Error()}
	}
	return gin.H{"message": err.Error()}
}

// SetContextError set the error to Context,and the error will be handled by ErrorHandleMiddleware
func SetContextError(c *gin.Context, errCode int, err error) {
	ce := c.Error(err)
	ce.Type = gin.ErrorType(errCode)
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
	*em.config = DefaultErrorHandleConfig
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
	var messageError error
	if em.config.Message != "" {
		messageError = errors.New(em.config.Message)
	}
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			code, e := em.config.ErrorParser(c, messageError)
			if c.Writer.Written() { // if the status has been written,must not write again
				code = c.Writer.Status()
			}
			NegotiateResponse(c, code, e, em.config.NegotiateFormat)
		}
	}
}

func defaultErrorParser(c *gin.Context, public error) (int, any) {
	var errs = make([]gin.H, len(c.Errors))
	var code = c.Writer.Status()
	for i, e := range c.Errors {
		switch e.Type {
		case gin.ErrorTypePublic:
			errs[i] = FormatResponseError(0, e.Err)
		case gin.ErrorTypePrivate:
			if public == nil {
				errs[0] = FormatResponseError(http.StatusInternalServerError, e.Err)
			} else {
				errs[0] = FormatResponseError(http.StatusInternalServerError, public)
			}
		default:
			errs[i] = FormatResponseError(int(e.Type), e.Err)
		}
	}
	if code == http.StatusOK {
		code = http.StatusInternalServerError
	}
	return code, gin.H{"errors": errs}
}

func (em *ErrorHandleMiddleware) Shutdown(_ context.Context) error {
	return nil
}

// NegotiateResponse calls different Render according to acceptably Accept format.
//
// no support format described:
//
//	protobuf: gin.H is not protoreflect.ProtoMessage
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
		if c.Writer.Written() {
			c.Error(errors.New("the accepted formats are not offered by the server")) // nolint: errcheck
		} else {
			c.AbortWithError(http.StatusNotAcceptable, errors.New("the accepted formats are not offered by the server")) // nolint: errcheck
		}
	}
}
