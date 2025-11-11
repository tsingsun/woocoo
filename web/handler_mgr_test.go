package web

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/tsingsun/woocoo/web/handler/gzip"
)

type mockMiddleware struct{}

func newMckMiddleware() handler.Middleware {
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
		newFuncs    map[string]handler.MiddlewareNewFunc
		middlewares map[string]handler.Middleware
	}
	type args struct {
		name    string
		handler handler.MiddlewareNewFunc
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
				handler: handler.WrapMiddlewareApplyFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
					return func(c *gin.Context) {
						c.Set("test", 1)
						c.Next()
					}
				}),
			},
			fields: fields{
				newFuncs: map[string]handler.MiddlewareNewFunc{
					"test": handler.WrapMiddlewareApplyFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
						return func(c *gin.Context) {
							c.Next()
						}
					}),
				},
				middlewares: map[string]handler.Middleware{
					"test": newMckMiddleware(),
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &HandlerManager{
				newFuncs:    tt.fields.newFuncs,
				middlewares: tt.fields.middlewares,
			}
			m.Register(tt.args.name, tt.args.handler)
			if got, ok := m.Get(tt.args.name); ok {
				c := &gin.Context{
					Keys: make(map[any]any),
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
		newFuncs    map[string]handler.MiddlewareNewFunc
		middlewares map[string]handler.Middleware
	}
	type args struct {
		key string
		mid handler.Middleware
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
				newFuncs:    make(map[string]handler.MiddlewareNewFunc),
				middlewares: make(map[string]handler.Middleware),
			},
		},
		{
			name: "override panic",
			args: args{
				key: "gzip",
				mid: gzip.Gzip(),
			},
			fields: fields{
				newFuncs:    make(map[string]handler.MiddlewareNewFunc),
				middlewares: map[string]handler.Middleware{"gzip": gzip.Gzip()},
			},
			panic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &HandlerManager{
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
