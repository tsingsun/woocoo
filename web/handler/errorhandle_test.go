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
			e := NewErrorHandle()
			gin.SetMode(gin.ReleaseMode)
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
			gin.SetMode(gin.ReleaseMode)
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
			gin.SetMode(gin.ReleaseMode)
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
