package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewSimpleMiddleware(t *testing.T) {
	type args struct {
		name      string
		applyFunc MiddlewareApplyFunc
	}
	tests := []struct {
		name string
		args args
		cfg  *conf.Configuration
		want *SimpleMiddleware
	}{
		{
			name: "test",
			args: args{
				name: "test",
				applyFunc: func(cfg *conf.Configuration) gin.HandlerFunc {
					return func(c *gin.Context) {
						c.Next()
					}
				},
			},
			cfg:  conf.New(),
			want: &SimpleMiddleware{name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSimpleMiddleware(tt.args.name, tt.args.applyFunc)
			got.ApplyFunc(tt.cfg)
			assert.Equal(t, tt.want.Name(), got.Name())
		})
	}
}

func TestPathSkip(t *testing.T) {
	type args struct {
		list []string
		url  *url.URL
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "start slash",
			args: args{
				list: []string{"/"},
				url:  &url.URL{Path: "/"},
			},
			want: true,
		},
		{
			name: "empty path",
			args: args{
				list: []string{"/"},
				url:  func() *url.URL { u, _ := url.Parse("http://www.example.com"); return u }(),
			},
			want: true,
		},
		{
			name: "no exist",
			args: args{
				list: []string{"/abc"},
				url:  &url.URL{Path: "/ab"},
			},
			want: false,
		},
		{
			name: "empty list",
			args: args{
				list: []string{},
				url:  &url.URL{Path: "/ab"},
			},
			want: false,
		},
		{
			name: "end splash",
			args: args{
				list: []string{"/abc"},
				url:  &url.URL{Path: "/abc/"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, PathSkip(StringsToMap(tt.args.list), tt.args.url), "PathSkip(%v, %v)", tt.args.list, tt.args.url)
		})
	}
}

func TestDerivativeContext(t *testing.T) {
	t.Run("git nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		assert.IsType(t, &gin.Context{}, ctx)
	})
	t.Run("set", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		SetDerivativeContext(ctx, context.Background())
		got := GetDerivativeContext(ctx)
		assert.Equal(t, context.Background(), got)
	})
	t.Run("try", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		SetDerivativeContext(ctx, context.Background())
		ok := DerivativeContextWithValue(ctx, "test", "testval")
		assert.True(t, ok)
		got := GetDerivativeContext(ctx)
		assert.Equal(t, "testval", got.Value("test"))
	})
}

func TestWrapMiddlewareApplyFunc(t *testing.T) {
	wf := WrapMiddlewareApplyFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	})
	mw := wf()
	assert.Equal(t, "test", mw.Name())
}
