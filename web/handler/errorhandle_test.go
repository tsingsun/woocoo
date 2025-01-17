package handler

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorHandleMiddleware_ApplyFunc(t *testing.T) {
	mkerr := errors.New("mock error")
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name   string
		handle gin.HandlerFunc
		args   args
		check  func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware)
	}{
		{
			name: "default",
			handle: func(c *gin.Context) {
				c.Error(mkerr) //nolint:errcheck
			},
			args: args{cfg: conf.NewFromStringMap(nil)},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Equal(t, http.StatusInternalServerError, r.Code)
				assert.Contains(t, r.Body.String(), mkerr.Error())
			},
		},
		{
			name: "written header",
			handle: func(c *gin.Context) {
				c.AbortWithStatus(http.StatusCreated)
				c.Error(mkerr)
			},
			args: args{cfg: conf.NewFromStringMap(nil)},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Equal(t, http.StatusCreated, r.Code)
				assert.Contains(t, r.Body.String(), mkerr.Error())
			},
		},
		{
			name: "keep custom status",
			handle: func(c *gin.Context) {
				c.Status(http.StatusCreated)
				c.Error(mkerr)
			},
			args: args{cfg: conf.NewFromStringMap(nil)},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Equal(t, http.StatusCreated, r.Code)
			},
		},
		{
			name: "gin error type",
			handle: func(c *gin.Context) {
				ge := c.Error(mkerr)
				ge.Type = 0
			},
			args: args{cfg: conf.New()},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Equal(t, http.StatusInternalServerError, r.Code)
				assert.NotContains(t, r.Body.String(), "code")
			},
		},
		{
			name: "public error will be shown",
			args: args{cfg: conf.NewFromStringMap(map[string]any{
				"message": "public message can be shown",
			})},
			handle: func(c *gin.Context) {
				ge := c.Error(mkerr)
				ge.Type = gin.ErrorTypePublic
			},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Equal(t, http.StatusInternalServerError, r.Code)
				assert.Contains(t, r.Body.String(), mkerr.Error())
				assert.NotContains(t, r.Body.String(), "public message can be shown")
			},
		},
		{
			name: "deny private error",
			args: args{cfg: conf.NewFromStringMap(map[string]any{
				"message": "public message can be shown",
			})},
			handle: func(c *gin.Context) {
				c.Error(mkerr)
			},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Equal(t, http.StatusInternalServerError, r.Code)
				assert.NotContains(t, r.Body.String(), mkerr.Error())
				assert.Contains(t, r.Body.String(), "public message can be shown")
			},
		},
		{
			name: "negotiate", handle: func(c *gin.Context) {
				c.Error(mkerr) //nolint:errcheck
			}, args: args{cfg: conf.NewFromStringMap(map[string]any{
				"name":    "Negotiate",
				"accepts": strings.Join([]string{binding.MIMEJSON, binding.MIMEMSGPACK2}, ","),
			})},
			check: func(t *testing.T, r *httptest.ResponseRecorder, middleware *ErrorHandleMiddleware) {
				assert.Len(t, middleware.config.NegotiateFormat, 2)
				assert.Equal(t, http.StatusInternalServerError, r.Code)
				assert.Contains(t, r.Body.String(), mkerr.Error())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorHandle().(*ErrorHandleMiddleware)
			assert.Equal(t, e.Name(), ErrorHandlerName)
			web := gin.New()
			got := e.ApplyFunc(tt.args.cfg)
			web.Use(got)
			web.GET("/", tt.handle)
			req := httptest.NewRequest("GET", "/", nil)
			res := httptest.NewRecorder()
			web.ServeHTTP(res, req)
			tt.check(t, res, e)
		})
	}
}

func TestSetContextError(t *testing.T) {
	e := ErrorHandleMiddleware{
		config: new(ErrorHandleConfig),
	}
	eh := e.ApplyFunc(conf.NewFromParse(conf.NewParserFromStringMap(nil)))
	type args struct {
		code int
		err  error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "standError", args: args{
				code: http.StatusInternalServerError,
				err:  fmt.Errorf("standError"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			srv := gin.New()
			srv.Use(eh)
			srv.GET("/", func(c *gin.Context) {
				c.Error(errors.New("standError"))
				SetContextError(c, tt.args.code, tt.args.err)
			})
			srv.ServeHTTP(w, r)
			assert.Equal(t, tt.args.code, w.Code)
		})
	}
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	type field struct {
		midCfg *conf.Configuration
	}
	type args struct {
		accept string
		err    error
	}
	tests := []struct {
		name            string
		field           field
		args            args
		wantCode        int
		wantContentType string
	}{
		{
			name: "default",
			field: field{
				midCfg: conf.NewFromParse(conf.NewParserFromStringMap(nil)),
			},
			args: args{
				err:    fmt.Errorf("standError"),
				accept: binding.MIMEJSON,
			},
			wantCode:        http.StatusInternalServerError,
			wantContentType: binding.MIMEJSON,
		},
		{
			name: "proto",
			field: field{
				midCfg: conf.NewFromParse(conf.NewParserFromStringMap(nil)),
			},
			args: args{
				err:    fmt.Errorf("standError"),
				accept: binding.MIMEPROTOBUF,
			},
			wantCode:        http.StatusNotAcceptable,
			wantContentType: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorHandleMiddleware{}
			eh := e.ApplyFunc(tt.field.midCfg)

			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Accept", tt.args.accept)
			w := httptest.NewRecorder()

			srv := gin.New()
			srv.Use(eh)
			srv.GET("/", func(c *gin.Context) {
				SetContextError(c, tt.wantCode, tt.args.err)
			})
			srv.ServeHTTP(w, r)
			assert.Equal(t, tt.wantCode, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), tt.wantContentType)
		})
	}
}

func TestCustomErrorFormater(t *testing.T) {
	const (
		ErrBodyNotAllowed = "username/password not correct"
		ErrPrivateMasked  = "private mask"
	)

	customCodeMap := map[int]string{
		10000: "miss required param",
		10001: "invalid param",
	}

	customErrorMap := map[string]string{
		http.ErrBodyNotAllowed.Error(): ErrBodyNotAllowed,
		http.ErrMissingFile.Error():    "miss required file",
		errors.New("private").Error():  ErrPrivateMasked,
	}
	type args struct {
		customCodeMap  map[int]string
		customErrorMap map[string]string
	}
	tests := []struct {
		name    string
		args    args
		check   func(t *testing.T, hf http.HandlerFunc)
		wantErr string
	}{
		{
			name: "empty",
			args: args{
				customCodeMap:  nil,
				customErrorMap: nil,
			},
			check: func(t *testing.T, hf http.HandlerFunc) {
				assert.HTTPBodyContains(t, hf, "GET", "/", nil, "abc")
				assert.HTTPBodyContains(t, hf, "GET", "/p", nil, "private")
			},
		},
		{
			name: "custom",
			args: args{
				customCodeMap:  customCodeMap,
				customErrorMap: customErrorMap,
			},
			check: func(t *testing.T, hf http.HandlerFunc) {
				assert.HTTPBodyContains(t, hf, "GET", "/", nil, "abc")
				assert.HTTPBodyContains(t, hf, "GET", "/p", nil, ErrPrivateMasked)
			},
		},
	}
	for _, tt := range tests {
		SetErrorMap(tt.args.customCodeMap, tt.args.customErrorMap)
		e := ErrorHandleMiddleware{}
		eh := e.ApplyFunc(conf.New())
		srv := gin.New()
		srv.Use(eh)
		srv.GET("/", func(c *gin.Context) {
			SetContextError(c, int(gin.ErrorTypePublic), errors.New("abc"))
		})
		srv.GET("/p", func(c *gin.Context) {
			SetContextError(c, int(gin.ErrorTypePrivate), errors.New("private"))
		})
		tt.check(t, srv.ServeHTTP)
	}
}

func TestCustomErrorHandler(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	hdl := NewErrorHandle(WithMiddlewareConfig(func(config any) {
		codeMap := map[uint64]any{
			10000: "miss required param",
			10001: "invalid param",
		}
		errorMap := map[interface{ Error() string }]string{
			http.ErrBodyNotAllowed: "username/password not correct",
			http.ErrMissingFile:    "miss required file",
		}
		c := config.(*ErrorHandleConfig)
		c.Accepts = "application/json,application/xml"
		c.Message = "internal error"
		c.ErrorParser = func(c *gin.Context, public error) (int, any) {
			var errs = make([]gin.H, len(c.Errors))
			for i, e := range c.Errors {
				if txt, ok := codeMap[uint64(e.Type)]; ok {
					errs[i] = gin.H{"code": i, "message": txt}
					continue
				}
				if txt, ok := errorMap[e.Err]; ok {
					errs[i] = gin.H{"code": i, "message": txt}
					continue
				}
				errs[i] = gin.H{"code": e.Type, "message": e.Error()}
			}
			return 0, errs
		}
	}))

	tests := []struct {
		name  string
		cfg   *conf.Configuration
		code  int
		err   error
		check func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name: "with-config",
			cfg:  conf.New(),
			code: http.StatusForbidden,
			err:  errors.New("username/password not correct"),
			check: func(t *testing.T, r *httptest.ResponseRecorder) {
				assert.Contains(t, r.Body.String(), "username/password not correct")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := gin.New()
			srv.Use(hdl.ApplyFunc(tt.cfg))
			srv.GET("/", func(c *gin.Context) {
				SetContextError(c, tt.code, tt.err)
			})
			r := httptest.NewRecorder()
			srv.ServeHTTP(r, httptest.NewRequest("GET", "/", nil))
			tt.check(t, r)
		})
	}
}
