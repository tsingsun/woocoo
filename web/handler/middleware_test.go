package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
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
			want: &SimpleMiddleware{name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSimpleMiddleware(tt.args.name, tt.args.applyFunc)
			assert.Equal(t, tt.want.Name(), got.Name())
		})
	}
}
