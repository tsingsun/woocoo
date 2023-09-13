package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
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

type mockMiddleware struct{}

func newMckMiddleware() Middleware {
	return &mockMiddleware{}
}

func (m *mockMiddleware) Name() string {
	return "mock"
}

func (m *mockMiddleware) ApplyFunc(_ *conf.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func (m *mockMiddleware) Shutdown(_ context.Context) error {
	return nil
}

func TestManager_Register(t *testing.T) {
	log.InitGlobalLogger()
	type fields struct {
		newFuncs    map[string]MiddlewareNewFunc
		middlewares map[string]Middleware
	}
	type args struct {
		name    string
		handler MiddlewareNewFunc
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   any
	}{
		{
			name: "override",
			args: args{
				name: "test",
				handler: WrapMiddlewareApplyFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
					return func(c *gin.Context) {
						c.Set("test", 1)
						c.Next()
					}
				}),
			},
			fields: fields{
				newFuncs: map[string]MiddlewareNewFunc{
					"test": WrapMiddlewareApplyFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
						return func(c *gin.Context) {
							c.Next()
						}
					}),
				},
				middlewares: map[string]Middleware{
					"test": newMckMiddleware(),
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				newFuncs:    tt.fields.newFuncs,
				middlewares: tt.fields.middlewares,
			}
			m.Register(tt.args.name, tt.args.handler)
			if got, ok := m.Get(tt.args.name); ok {
				c := &gin.Context{
					Keys: make(map[string]any),
				}
				got().ApplyFunc(conf.New())(c)
				assert.Equal(t, tt.want, c.GetInt(tt.args.name))
			}
			assert.NoError(t, m.Shutdown(context.Background()))
		})
	}
}

func TestManager_RegisterMiddleware(t *testing.T) {
	type fields struct {
		newFuncs    map[string]MiddlewareNewFunc
		middlewares map[string]Middleware
	}
	type args struct {
		key string
		mid Middleware
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		panic  bool
	}{
		{
			name: "normal",
			args: args{
				key: "test",
				mid: newMckMiddleware(),
			},
			fields: fields{
				newFuncs:    make(map[string]MiddlewareNewFunc),
				middlewares: make(map[string]Middleware),
			},
		},
		{
			name: "override panic",
			args: args{
				key: "gzip",
				mid: Gzip(),
			},
			fields: fields{
				newFuncs:    make(map[string]MiddlewareNewFunc),
				middlewares: map[string]Middleware{"gzip": Gzip()},
			},
			panic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				newFuncs:    tt.fields.newFuncs,
				middlewares: tt.fields.middlewares,
			}
			if tt.panic {
				assert.Panics(t, func() {
					m.RegisterMiddleware(tt.args.key, tt.args.mid)
				})
				return
			}
			m.RegisterMiddleware(tt.args.key, tt.args.mid)
			m.GetMiddleware(tt.args.key)
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
			assert.Equalf(t, tt.want, PathSkip(tt.args.list, tt.args.url), "PathSkip(%v, %v)", tt.args.list, tt.args.url)
		})
	}
}

func TestGetMiddlewareKey(t *testing.T) {
	type args struct {
		group string
		name  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				group: "test",
				name:  "test",
			},
			want: "test:test",
		},
		{
			name: "empty",
			args: args{
				group: "/",
				name:  "test",
			},
			want: "default:test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetMiddlewareKey(tt.args.group, tt.args.name), "GetMiddlewareKey(%v, %v)", tt.args.group, tt.args.name)
		})
	}
}
