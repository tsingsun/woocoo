package gql

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/authz"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	"github.com/vektah/gqlparser/v2/ast"
)

var gqlSchemaMock = graphql.ExecutableSchemaMock{
	ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]any) (int, bool) {
		panic("mock out the Complexity method")
	},
	ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
		panic("mock out the Exec method")
	},
	SchemaFunc: func() *ast.Schema {
		// panic("mock out the Schema method")
		return nil
	},
}

func TestRegistrySchema(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - graphql:
      - graphql:
          basePath: "/graphql"
          middlewares:
            - graphql:
                queryPath: "/query"
                docPath: "/doc"
                group: "/graphql"
                header:
                  Authorization: "Bearer 123456"
                  X-Tenant-Id: "1"
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.WithConfiguration(cfg.Sub("web")), web.RegisterMiddleware(New()))
	gqlsrvList, err := RegisterSchema(srv, &gqlSchemaMock, &gqlSchemaMock)
	require.NoError(t, err)
	if !assert.Len(t, gqlsrvList, 2) {
		return
	}

	t.Run("test default", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("test doc", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/graphql/doc", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"Authorization":"Bearer 123456","X-Tenant-Id":"1"`)
	})
	t.Run("test sub", func(t *testing.T) {
		g2 := gqlsrvList[0]
		r := httptest.NewRequest("GET", "/graphql", nil)
		w := httptest.NewRecorder()
		g2.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestCheckPermissions(t *testing.T) {
	var cfgStr = `
authz:
  autoSave: false
  model: |
    [request_definition]
    r = sub, obj, act
    [policy_definition]
    p = sub, obj, act
    [role_definition]
    g = _, _
    [policy_effect]
    e = some(where (p.eft == allow))
    [matchers]
    m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act

web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - graphql:
                withAuthorization: true
`
	authz.SetAdapter(stringadapter.NewAdapter(`p, 1, "hello", POST`))

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.WithConfiguration(cfg.Sub("web")), web.RegisterMiddleware(New()))
	expectedPanic := "gql panic"
	expectedPanicErr := errors.New("gql panic error")
	mock := graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			return func(ctx context.Context) *graphql.Response {
				gctx, _ := FromIncomingContext(ctx)
				if _, ok := gctx.Get("panic"); ok {
					panic(expectedPanic)
				}
				if _, ok := gctx.Get("panicerr"); ok {
					panic(expectedPanicErr)
				}
				return &graphql.Response{
					Data: []byte("{}"),
				}
			}
		},
		SchemaFunc: func() *ast.Schema {
			return &ast.Schema{
				Query: &ast.Definition{
					Kind: ast.Object,
					Name: "Query",
					Fields: []*ast.FieldDefinition{
						{
							Name:     "hello",
							Type:     ast.NamedType("Boolean", &ast.Position{}),
							Position: &ast.Position{},
						},
					},
				},
				Types: map[string]*ast.Definition{
					"Boolean": {
						Kind:     ast.Scalar,
						Name:     "Boolean",
						Position: &ast.Position{},
					},
				},
			}
		},
	}

	gqlsrv, err := RegisterSchema(srv, &mock)
	require.NoError(t, err)
	t.Run("allow", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest("POST", "/graphql/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`))).
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		gqlsrv[0].ServeHTTP(w, r)
		if !assert.Equal(t, http.StatusOK, w.Code) {
			t.Log(w.Body.String())
		}
	})
	t.Run("panic string", func(t *testing.T) {
		conf.Global().Development = true
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest("POST", "/graphql/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`))).
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		c.Set("panic", true)
		gqlsrv[0].ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), expectedPanic)
	})
	t.Run("panic err", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest("POST", "/graphql/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`))).
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		c.Set("panicerr", true)
		gqlsrv[0].ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), expectedPanicErr.Error())
	})
	t.Run("reject", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest("POST", "/graphql/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`))).
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(map[string]any{"sub": "2"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		gqlsrv[0].ServeHTTP(w, r)
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.Body.String(), "action hello not allowed")
		}
	})
}

func TestHandler_ApplyFunc(t *testing.T) {
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name  string
		args  args
		check func(*Handler)
		panic bool
	}{
		{
			name: "header",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
queryPath: "/query"
docPath: "/doc"
group: "/graphql"
header: 
  Authorization: "Bearer 123456"
  X-Tenant-Id: "1"
`)),
			},
			check: func(handler *Handler) {
				assert.Equal(t, "Bearer 123456", handler.opts[0].DocHeader["Authorization"])
			},
		},
		{
			name: "Authorization config incorrect",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
withAuthorization: true
`)),
			},
			panic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			if tt.panic {
				assert.Panics(t, func() {
					h.ApplyFunc(tt.args.cfg)
				})
				return
			}
			h.ApplyFunc(tt.args.cfg)
			if tt.check != nil {
				tt.check(h)
			}
		})
	}
}
