package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
	"strings"
)

var (
	DefaultNegotiateFormat = []string{binding.MIMEJSON}
)

type (
	// ErrorHandleConfig is the config for error handle
	ErrorHandleConfig struct {
		// NegotiateFormat is the offered http media type format for errors.
		// formats split by comma, such as "application/json,application/xml"
		Accepts         string      `json:"accepts" yaml:"accepts"`
		NegotiateFormat []string    `json:"-" yaml:"-"`
		ErrorParser     ErrorParser `json:"-" yaml:"-"`
		// Message is the message you want mask all private error, it will not mask public error.
		// see gin.ErrorType
		Message string `json:"message" yaml:"message"`
	}

	// ErrorParser is the error parser, public error adopt by private error to show to the client.
	ErrorParser func(c *gin.Context, public error) (int, any)

	// ErrorHandleMiddleware is the middleware for error handle to format the errors to client
	ErrorHandleMiddleware struct {
		config *ErrorHandleConfig
		opts   MiddlewareOptions
	}
)

// NewErrorHandle is the error handle middleware
func NewErrorHandle(opts ...MiddlewareOption) *ErrorHandleMiddleware {
	mw := &ErrorHandleMiddleware{
		config: new(ErrorHandleConfig),
	}
	mipts := NewMiddlewareOption(opts...)
	if mipts.ConfigFunc != nil {
		mipts.ConfigFunc(mw.config)
	}
	return mw
}

// ErrorHandle is the error handle middleware apply function. see MiddlewareNewFunc
func ErrorHandle() Middleware {
	mw := NewErrorHandle()
	return mw
}

func (mw *ErrorHandleMiddleware) Name() string {
	return ErrorHandlerName
}

func (mw *ErrorHandleMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if err := cfg.Unmarshal(&mw.config); err != nil {
		panic(err)
	}
	if mw.config.Accepts != "" {
		mw.config.NegotiateFormat = strings.Split(mw.config.Accepts, ",")
	} else if len(mw.config.NegotiateFormat) == 0 {
		mw.config.NegotiateFormat = DefaultNegotiateFormat
	}
	var messageError error
	if mw.config.Message != "" {
		messageError = errors.New(mw.config.Message)
	}
	if mw.config.ErrorParser == nil {
		mw.config.ErrorParser = DefaultErrorParser
	}
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			code, e := mw.config.ErrorParser(c, messageError)
			NegotiateResponse(c, code, e, mw.config.NegotiateFormat)
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
			errs[i] = DefaultErrorFormater(code, e.Err)
		case gin.ErrorTypePrivate:
			if public == nil {
				errs[i] = DefaultErrorFormater(code, e.Err)
			} else {
				errs[i] = DefaultErrorFormater(code, public)
			}
		default:
			errs[i] = DefaultErrorFormater(int(e.Type), e.Err)
		}
	}
	return code, gin.H{"errors": errs}
}

// DefaultErrorFormater is the default error formater
var DefaultErrorFormater = FormatResponseError

var (
	errorCodeMap    = map[int]string{}
	errorMap        = map[string]string{}
	useErrorCodeMap bool
	useErrorMap     bool
)

// SetErrorMap set the error code map and error message map for ErrorHandler.
func SetErrorMap(cm map[int]string, em map[string]string) {
	if cm != nil {
		errorCodeMap = cm
		useErrorCodeMap = len(errorCodeMap) > 0
	}
	if em != nil {
		errorMap = em
		useErrorMap = len(errorMap) > 0
	}
}

// FormatResponseError converts a http error to gin.H
func FormatResponseError(code int, err error) gin.H {
	code, txt := LookupErrorCode(code, err)
	if code != 0 {
		return gin.H{
			"code":    code,
			"message": txt,
		}
	}
	return gin.H{
		"message": txt,
	}
}

// LookupErrorCode lookup error code and customer message where you want to mask the error.
func LookupErrorCode(code int, err error) (int, string) {
	if useErrorCodeMap {
		if txt, ok := errorCodeMap[code]; ok {
			return code, txt
		}
	}
	if useErrorMap {
		if txt, ok := errorMap[err.Error()]; ok {
			return code, txt
		}
	}
	if code > 0 {
		return code, err.Error()
	}
	return 0, err.Error()
}

// NegotiateResponse calls different Render according to acceptably Accept format.
//
// HTTP Status Code behavior:
//
// 1. response has sent cannot change it.
// 2. if code is not default, use it.
// 3. if code is default, use passed code.
//
// no support format described:
//
//	protobuf: gin.H is not protoreflect.ProtoMessage
func NegotiateResponse(c *gin.Context, code int, data any, offered []string) {
	if c.Writer.Written() { // if the status has been written, must not write again
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

// SetContextError set the error to Context, and the error will be handled by ErrorHandleMiddleware
func SetContextError(c *gin.Context, code int, err error) {
	ce := c.Error(err)
	ce.Type = gin.ErrorType(code)
}

// AbortWithError gin context's AbortWithError prevent to reset response header, so we need to replace it.
func AbortWithError(c *gin.Context, code int, err error) {
	c.Abort()
	c.Error(err) // nolint: errcheck
	c.Status(code)
}
