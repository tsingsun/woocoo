package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
)

type (
	// ErrorHandleConfig is the config for error handle
	ErrorHandleConfig struct {
		ErrorParser ErrorParser
	}

	// ErrorParser is the error parser
	ErrorParser func(c *gin.Context) (int, interface{})
)

var defaultErrorHandleConfig = &ErrorHandleConfig{
	ErrorParser: defaultErrorParser,
}

func defaultErrorParser(c *gin.Context) (int, interface{}) {
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
	return code, errs
}

// FormatResponseError converts a http error to gin.H
func FormatResponseError(code int, err error) gin.H {
	if code != 0 {
		return gin.H{"code": code, "msg": err.Error()}
	}
	return gin.H{"msg": err.Error()}
}

// SetContextError set the error to Context,and the error will be handled by ErrorHandleMiddleware
func SetContextError(c *gin.Context, code int, err error) {
	ce := c.Error(err)
	ce.Type = gin.ErrorType(code)
	c.Status(code)
}

// ErrorHandleMiddleware is the middleware for error handle to format the errors to client
type ErrorHandleMiddleware struct {
}

// ErrorHandle is the error handle middleware
func ErrorHandle() *ErrorHandleMiddleware {
	return &ErrorHandleMiddleware{}
}

func (e ErrorHandleMiddleware) Name() string {
	return "errorHandle"
}

func (e ErrorHandleMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	eh := defaultErrorHandleConfig
	if err := cfg.Unmarshal(&eh); err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			code, e := eh.ErrorParser(c)
			c.JSON(code, e)
		}
	}
}

func (e ErrorHandleMiddleware) Shutdown() {
}
