package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/cache/redisc"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
)

func TestKeyAuth(t *testing.T) {
	mredis := miniredis.RunT(t)
	err := cache.RegisterCache("keyAuthStore", func() cache.Cache {
		return redisc.New(conf.NewFromStringMap(map[string]any{
			"type":  "standalone",
			"addrs": []string{mredis.Addr()},
		}))
	}())
	require.NoError(t, err)
	type args struct {
		cfg  *conf.Configuration
		opts []handler.MiddlewareOption
	}
	tests := []struct {
		name       string
		args       args
		request    *http.Request
		wantStatus int
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "default",
			args: args{
				cfg:  conf.NewFromStringMap(map[string]any{}),
				opts: []handler.MiddlewareOption{},
			},
			request: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-API-KEY", "secret")
				return req
			}(),
			wantStatus: 500,
			wantErr:    assert.NoError,
		},
		{
			name: "header",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{}),
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func() any {
						return &handler.KeyAuthConfig{
							Validator: func(c *gin.Context, keyAuth string) (bool, error) {
								if keyAuth == "secret" {
									return true, nil
								}
								return false, errors.New("invalid key")
							},
						}
					}),
				},
			},
			request: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-API-KEY", "secret")
				return req
			}(),
			wantStatus: 200,
			wantErr:    assert.NoError,
		},
		{
			name: "header-error",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{}),
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func() any {
						return &handler.KeyAuthConfig{
							KeyLookup: "header:API-KEY",
							Validator: func(c *gin.Context, keyAuth string) (bool, error) {
								return false, errors.New("invalid key")
							},
						}
					}),
				},
			},
			request: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("API-KEY", "secret1")
				return req
			}(),
			wantStatus: 401,
			wantErr:    assert.NoError,
		},
		{
			name: "errorHandler",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{}),
				opts: []handler.MiddlewareOption{
					handler.WithMiddlewareConfig(func() any {
						return &handler.KeyAuthConfig{
							KeyLookup: "form:API-KEY",
							Validator: func(c *gin.Context, keyAuth string) (bool, error) {
								assert.Equal(t, "secret1", keyAuth)
								return false, errors.New("invalid key")
							},
							ErrorHandler: func(c *gin.Context, err error) error {
								return nil
							},
						}
					}),
				},
			},
			request: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Form = url.Values{}
				req.Form.Set("API-KEY", "secret1")
				return req
			}(),
			wantStatus: 200,
			wantErr:    assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := handler.KeyAuth(tt.args.opts...)
			defer mw.Shutdown(context.TODO())
			assert.NotNil(t, mw)
			srv := web.New()
			if len(tt.args.opts) > 0 {
				srv.Router().Engine.Use(mw.ApplyFunc(tt.args.cfg))
			} else {
				assert.Panics(t, func() {
					mw.ApplyFunc(tt.args.cfg)
				})
				return
			}
			srv.Router().Engine.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "")
			})
			r := tt.request
			var w = httptest.NewRecorder()
			srv.Router().ServeHTTP(w, r)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
