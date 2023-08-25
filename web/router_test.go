package web

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
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

func TestRouter_Apply(t *testing.T) {
	type fields struct {
		Router *Router
	}
	type args struct {
		cnf *conf.Configuration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
		check   func(r *Router)
	}{
		{
			name: "test",
			fields: fields{
				Router: NewRouter(&ServerOptions{}),
			},
			args: args{
				cnf: conf.NewFromStringMap(map[string]any{
					"routerGroups": []any{
						map[string]any{
							"default": map[string]any{
								"basePath": "/",
							},
						},
						map[string]any{
							"sub2": map[string]any{
								"basePath": "/sub1/sub2",
							},
						},
						map[string]any{
							"sub1": map[string]any{
								"basePath": "/sub1",
							},
						},
					},
				}),
			},
			wantErr: assert.NoError,
			check: func(r *Router) {
				assert.Equal(t, 3, len(r.Groups))
				assert.Equal(t, "/", r.Groups[0].basePath)
				assert.Equal(t, "/sub1/sub2", r.Groups[1].basePath)
				assert.Equal(t, "/sub1", r.Groups[2].basePath)
				assert.Equal(t, 0, len(r.Groups[0].Group.Handlers))
				assert.Equal(t, 0, len(r.Groups[1].Group.Handlers))
				assert.Equal(t, 0, len(r.Groups[2].Group.Handlers))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.Router
			assert.NoError(t, r.Apply(tt.args.cnf))
			tt.check(r)
		})
	}
}
