package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
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
			got.Shutdown(context.Background())
		})
	}
}

func TestManager(t *testing.T) {
	log.InitGlobalLogger()
	type fields struct {
		middlewares map[string]Middleware
	}
	type args struct {
		name    string
		handler Middleware
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{
			name: "override",
			args: args{
				name: "test",
				handler: NewSimpleMiddleware("test", func(cfg *conf.Configuration) gin.HandlerFunc {
					return func(c *gin.Context) {
						c.Set("test", 1)
						c.Next()
					}
				}),
			},
			fields: fields{
				middlewares: map[string]Middleware{
					"test": NewSimpleMiddleware("test", func(cfg *conf.Configuration) gin.HandlerFunc {
						return func(c *gin.Context) {
							c.Next()
						}
					}),
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				middlewares: tt.fields.middlewares,
			}
			m.RegisterHandlerFunc(tt.args.name, tt.args.handler)
			if got, ok := m.Get(tt.args.name); ok {
				c := &gin.Context{
					Keys: make(map[string]any),
				}
				got.ApplyFunc(conf.New())(c)
				assert.Equal(t, tt.want, c.GetInt(tt.args.name))
				got.Shutdown(context.Background())
			}
		})
	}
}
