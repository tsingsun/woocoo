package web

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_AddRule(t *testing.T) {
	r := NewRouter(&ServerOptions{})
	r.Engine = gin.New()
	var newfunc gin.HandlerFunc = func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	}
	r.Engine.GET("/", newfunc)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Engine.ServeHTTP(rec, req)
	assert.Equal(t, rec.Body.String(), "hello")
}

func TestRouter_FindGroup(t *testing.T) {
	type fields struct {
		Engine        *gin.Engine
		Groups        []*RouterGroup
		serverOptions *ServerOptions
	}
	type args struct {
		basePath string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *RouterGroup
	}{
		{
			name: "find empty",
			fields: fields{
				Engine:        gin.New(),
				Groups:        []*RouterGroup{{basePath: "/"}},
				serverOptions: &ServerOptions{},
			},
			args: args{
				basePath: "",
			},
			want: &RouterGroup{basePath: "/"},
		},
		{
			name: "find",
			fields: fields{
				Engine:        gin.New(),
				Groups:        []*RouterGroup{{basePath: "/index"}, {basePath: "/gql"}},
				serverOptions: &ServerOptions{},
			},
			args: args{
				basePath: "/index",
			},
			want: &RouterGroup{basePath: "/index"},
		},
		{
			name: "not found",
			fields: fields{
				Engine:        gin.New(),
				Groups:        []*RouterGroup{{basePath: "/"}},
				serverOptions: &ServerOptions{},
			},
			args: args{
				basePath: "/xxx",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Router{
				Engine:        tt.fields.Engine,
				Groups:        tt.fields.Groups,
				serverOptions: tt.fields.serverOptions,
			}
			assert.Equalf(t, tt.want, r.FindGroup(tt.args.basePath), "FindGroup(%v)", tt.args.basePath)
		})
	}
}
