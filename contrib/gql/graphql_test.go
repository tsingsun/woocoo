package gql

import (
	"bytes"
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tsingsun/woocoo/pkg/log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

const (
	secretToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiI2N2E4NzQ4MmU5MWY0ZjJlOTIyMGY1MTM3NjE4NWI3ZSIsInN1YiI6IjEyMzQ1Njc4OTAiLCJuYW1lIjoiSm9obiBEb2UiLCJpYXQiOjE1MTYyMzkwMjJ9.ey-P5Kz9BKn0IsMuJd6egrwdi7uv34G2s85pmfVgTo0`
)

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
				assert.Equal(t, "Bearer 123456", handler.opts.DocHeader["Authorization"])
			},
		},
		{
			name: "Authorization config incorrect",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
withAuthorization: true
`)),
			},
			panic: func() bool {
				security.DefaultAuthorizer = nil
				return true
			}(),
		},
		{
			name: "webHandler",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
middlewares:
  operation:
  - jwt:
      signingMethod: "HS256"
      signingKey: "secret"
  - nil:
`)),
			},
			check: func(handler *Handler) {
				mids := handler.opts.Middlewares["operation"].([]any)
				_, ok := mids[0].(map[string]any)["jwt"]
				assert.True(t, ok)
				assert.Len(t, mids, 2)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			assert.Equal(t, "graphql", h.Name())
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

func TestRegistrySchema(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - first:
          basePath: "/first"
          middlewares:
            - graphql:
      - second:
          basePath: "/second"
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
	srv := web.New(web.WithConfiguration(cfg.Sub("web")),
		web.WithMiddlewareNewFunc(graphqlHandlerName, Middleware))
	gqlsrvList, err := RegisterSchema(srv, &gqlSchemaMock, &gqlSchemaMock)
	require.NoError(t, err)
	if !assert.Len(t, gqlsrvList, 2) {
		return
	}

	t.Run("test default", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/first/", nil)
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

func TestRegistrySchema_NoDoc(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - graphql:
                docPath: ""
      - graphql:
          basePath: "/graphql"
          middlewares:
            - graphql:
                group: "/graphql"
                docPath: "" 
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.WithConfiguration(cfg.Sub("web")),
		RegisterMiddleware())
	gqlsrvList, err := RegisterSchema(srv, &gqlSchemaMock, &gqlSchemaMock)
	require.NoError(t, err)
	if !assert.Len(t, gqlsrvList, 2) {
		return
	}

	t.Run("no-doc", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("sub-no-doc", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/graphql", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRegistrySchema_WebHandler(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - graphql:
                middlewares:
                  operation:
                  - nilmid:
                  - jwt:
                      signingMethod: "HS256"
                      signingKey: "secret"
                  - recovery: 
                  response:
                  - accessLog:
                  - nilmid:
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.WithConfiguration(cfg.Sub("web")),
		web.WithMiddlewareNewFunc(graphqlHandlerName, Middleware))
	mock := graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			return func(ctx context.Context) *graphql.Response {
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
	gqlsrvList, err := RegisterSchema(srv, &mock)
	require.NoError(t, err)
	if !assert.Len(t, gqlsrvList, 1) {
		return
	}
	//gqlsrv  := gqlsrvList[0]

	t.Run("unauth", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("ok", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`)))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+secretToken)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

type mockAuthorizer struct {
	// user-role map
	users    map[string]string
	userNeed bool
	allowed  map[string]security.Action
}

func (m mockAuthorizer) Prepare(ctx context.Context, kind security.ArnKind, arnParts ...string) (*security.EvalArgs, error) {
	user, ok := security.FromContext(ctx)
	if !ok && m.userNeed {
		return nil, errors.New("security.IsAllow: user not found in context")
	}
	args := security.EvalArgs{
		User: user,
	}
	switch kind {
	case security.ArnKindGql:
		args.Action = security.Action(security.Resource(strings.Join(append(arnParts[:1], arnParts[2:]...), ":")))
	default:
		args.Action = security.Action(security.Resource(strings.Join(arnParts, ":")))
	}
	return &args, nil
}

func (m mockAuthorizer) Eval(ctx context.Context, args *security.EvalArgs) (bool, error) {
	if args.User.Identity().Name() == "2" {
		return false, nil
	}
	if len(m.allowed) > 0 {
		action := m.allowed[string(args.Action)]

		return action.MatchResource(string(args.Action)), nil
	}
	return false, nil
}

func (m mockAuthorizer) QueryAllowedResourceConditions(context.Context, *security.EvalArgs) ([]string, error) {
	panic("not used in this test")
}

func TestCheckPermissions(t *testing.T) {
	log.InitGlobalLogger()
	security.SetDefaultAuthorizer(&mockAuthorizer{
		allowed: map[string]security.Action{
			"test:hello": security.Action(`test:hello`),
		},
	})
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
    - default:
        middlewares:
        - graphql:
            withAuthorization: true
            appCode: "test"
`

	cfg := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	srv := web.New(web.WithConfiguration(cfg.Sub("web")),
		web.WithMiddlewareNewFunc(graphqlHandlerName, Middleware))
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
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "1"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		c.Set(security.PrincipalContextKey, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "1"}))
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
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "1"})))
		r.Header.Set("Content-Type", "application/json")
		c.Request = r
		c.Set(security.PrincipalContextKey, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "1"}))
		c.Set("panic", true)
		gqlsrv[0].ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), expectedPanic)
	})
	t.Run("panic err", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest("POST", "/graphql/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`))).
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "1"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		c.Set(security.PrincipalContextKey, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "1"}))
		c.Set("panicerr", true)
		gqlsrv[0].ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), expectedPanicErr.Error())
	})
	t.Run("reject", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest("POST", "/graphql/query", bytes.NewReader([]byte(`{"query":"query hello { hello() }"}`))).
			WithContext(security.WithContext(c, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "2"})))
		r.Header.Set("Content-Type", "application/json")

		c.Request = r
		c.Set(security.PrincipalContextKey, security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": "2"}))
		gqlsrv[0].ServeHTTP(w, r)
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.Body.String(), "action hello not allowed")
		}
	})
}
