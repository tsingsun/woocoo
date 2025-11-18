package gql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type MockResponse struct {
	Name string `json:"name"`
}

func (mr *MockResponse) UnmarshalGQL(v any) error {
	return nil
}

func (mr *MockResponse) MarshalGQL(w io.Writer) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(mr)
	if err != nil {
		panic(err)
	}

	ba := bytes.NewBuffer(bytes.TrimRight(buf.Bytes(), "\n"))

	fmt.Fprint(w, ba)
}

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
docHeader: 
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
                docHeader:
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
		r := httptest.NewRequest("POST", "/query", bytes.NewReader([]byte(`{"query":"query hello { hello }"}`)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, r)
		if !assert.Equal(t, http.StatusUnauthorized, w.Code) {
			t.Log(w.Body.String())
		}
	})
	t.Run("ok", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/query", bytes.NewReader([]byte(`{"query":"query hello { hello }"}`)))
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
	if args.User.Identity().Name() == "3" {
		return false, errors.New("mock error")
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
			"test:hello": `test:hello`,
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
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			hello: Boolean!
		}
		type Mutation {
			name: String!
		}
		type Subscription {
			name: String!
		}
		enum EnumType{
			"""Description for VALUE1"""
  			VALUE1
			"""Description for VALUE2"""
  			VALUE2
		}
	`})
	mock := graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			return func(ctx context.Context) *graphql.Response {
				gctx, _ := FromIncomingContext(ctx)
				if ps := gctx.Request.Header.Get("panic"); ps != "" {
					panic(expectedPanic)
				}
				if ps := gctx.Request.Header.Get("panicerr"); ps != "" {
					panic(expectedPanicErr)
				}
				if ps := gctx.Request.Header.Get("Type-Query"); ps != "" {
					return &graphql.Response{
						Data: []byte(`{"__type":{"name":"EnumType","enumValues":[{"name":"VALUE1","description":"Description for VALUE1"},{"name":"VALUE2","description":"Description for VALUE2"}]}}`),
					}
				}
				return &graphql.Response{
					Data: []byte("{}"),
				}
			}
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	}

	gqlsrv, err := RegisterSchema(srv, &mock)
	require.NoError(t, err)
	var reuqest = func(target, uid string) *http.Request {
		r := httptest.NewRequest("POST", target, bytes.NewReader([]byte(`{"query":"query hello { hello }"}`)))
		if uid != "" {
			r = r.WithContext(security.WithContext(context.Background(), security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": uid})))
		}
		r.Header.Set("Content-Type", "application/json")
		return r
	}
	t.Run("allow", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("panic string", func(t *testing.T) {
		conf.Global().Development = true
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Set("panic", "1")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), expectedPanic)
	})
	t.Run("panic err", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Set("panicerr", "1")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), expectedPanicErr.Error())
	})
	t.Run("reject", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "2")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "action hello is not allowed")
	})
	t.Run("miss ctx", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "2")
		gqlsrv[0].ServeHTTP(w, r)
		assert.Contains(t, w.Body.String(), ErrMissGinContext.Error())
	})
	t.Run("allow err", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "3")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "mock error")
	})
	t.Run("type query", func(t *testing.T) {
		cli := client.New(srv.Router(), func(bd *client.Request) {
			bd.HTTP = bd.HTTP.WithContext(security.WithContext(context.Background(),
				security.NewGenericPrincipalByClaims(jwt.MapClaims{"sub": 1})))
			bd.HTTP.URL.Path = "/query"
			bd.HTTP.Header.Set("Type-Query", "1")

		})
		query := `
query EnumType{ 
  __type(name: "EnumType") {
    name,
    enumValues { 
      name, 
      description 
    } 
  } 
}
`
		var resp map[string]any
		err = cli.Post(query, &resp)
		require.NoError(t, err)
		assert.Equal(t,
			map[string]any{
				"name": "EnumType",
				"enumValues": []any{
					map[string]any{
						"name":        "VALUE1",
						"description": "Description for VALUE1",
					},
					map[string]any{
						"name":        "VALUE2",
						"description": "Description for VALUE2",
					},
				},
			}, resp["__type"])
	})
}

func Test_envResponseError(t *testing.T) {
	t.Run("websocket", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/query", nil)
		c.Request.Header.Set("Connection", "upgrade")
		c.Request.Header.Set("Upgrade", "websocket")
		h := envResponseError(c, gqlerror.List{gqlerror.Errorf("test")})
		res := h(c)
		assert.Len(t, res.Errors, 1)
	})
}

func TestRecovery(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
    - default:
        middlewares:
        - recovery:
        - errorHandle:
        - graphql:
`

	cfg := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	srv := web.New(web.WithConfiguration(cfg.Sub("web")),
		web.WithMiddlewareNewFunc(graphqlHandlerName, Middleware))
	expectedPanic := "gql panic"
	expectedPanicErr := errors.New("gql panic error")
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			hello: Boolean!
		}
		type Mutation {
			name: String!
		}
	`})
	mock := graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			return func(ctx context.Context) *graphql.Response {
				gctx, _ := FromIncomingContext(ctx)
				if ps := gctx.Request.Header.Get("panic"); ps != "" {
					panic(expectedPanic)
				}
				if ps := gctx.Request.Header.Get("panicerr"); ps != "" {
					panic(expectedPanicErr)
				}
				return &graphql.Response{
					Data: []byte("{}"),
				}
			}
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	}

	_, err := RegisterSchema(srv, &mock)
	require.NoError(t, err)
	var reuqest = func(target, uid string) *http.Request {
		r := httptest.NewRequest("POST", target, bytes.NewReader([]byte(`{"query":"query hello { hello }"}`)))
		r.Header.Set("Content-Type", "application/json")
		return r
	}
	t.Run("string panic", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Add("panic", "1")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var rt graphql.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rt), w.Body.String())
		assert.Len(t, rt.Errors, 1)
		assert.Equal(t, expectedPanic, rt.Errors[0].Message)
	})
	t.Run("err panic", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Add("panicerr", "1")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var rt graphql.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rt))
		assert.Len(t, rt.Errors, 1)
		assert.Equal(t, expectedPanicErr.Error(), rt.Errors[0].Message)
	})
}

func TestWorkWithErrorHandler(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
    - default:
        middlewares:
        - recovery:
        - accessLog:
        - errorHandle:
        - graphql:
`

	cfg := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	srv := web.New(web.WithConfiguration(cfg.Sub("web")),
		web.WithMiddlewareNewFunc(graphqlHandlerName, Middleware))
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			hello: Boolean!
		}
		type Mutation {
			name: String!
		}
	`})
	mock := graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]any) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			opCtx := graphql.GetOperationContext(ctx)
			switch opCtx.Operation.Operation {
			case ast.Query:
				ran := false
				return func(ctx context.Context) *graphql.Response {
					if ran {
						return nil
					}
					ran = true
					gctx, _ := FromIncomingContext(ctx)
					ps := gctx.Request.Header.Get("ginError")
					switch ps {
					case "0":
						graphql.AddError(ctx, errors.New("common error"))
					case "withMeta":
						graphql.AddError(ctx, &gin.Error{
							Err:  errors.New("gin error {{field}}"),
							Type: 50000, // custom code
							Meta: map[string]any{
								"field": "value",
							},
						})
					default:
						graphql.AddError(ctx, &gin.Error{
							Err:  errors.New("gin error"),
							Type: 10000, // custom code
						})
					}

					return &graphql.Response{
						Data: []byte(`null`),
					}
				}
			case ast.Mutation:
				return graphql.OneShot(graphql.ErrorResponse(ctx, "mutations are not supported"))
			case ast.Subscription:
				return graphql.OneShot(graphql.ErrorResponse(ctx, "subscription are not supported"))
			default:
				return graphql.OneShot(graphql.ErrorResponse(ctx, "unsupported GraphQL operation"))
			}
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	}
	_, err := RegisterSchema(srv, &mock)
	require.NoError(t, err)
	var reuqest = func(target, uid string) *http.Request {
		r := httptest.NewRequest("POST", target, bytes.NewReader([]byte(`{"query":"query hello { hello }"}`)))
		r.Header.Set("Content-Type", "application/json")
		return r
	}
	t.Run("common error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Add("ginError", "0")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		var rt graphql.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rt), w.Body.String())
		assert.Len(t, rt.Errors, 1)
		assert.EqualValues(t, "common error", rt.Errors[0].Message)
		assert.Nil(t, rt.Errors[0].Extensions)
	})
	handler.SetErrorMap(
		map[uint64]string{10000: "10000 error"},
		map[string]string{"custom error": "custom error"},
	)

	t.Run("gin error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Add("ginError", "1")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		var rt graphql.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rt), w.Body.String())
		assert.Len(t, rt.Errors, 1)
		assert.EqualValues(t, 10000, rt.Errors[0].Extensions["code"])
		assert.EqualValues(t, "10000 error", rt.Errors[0].Message)
	})
	t.Run("gin error with meta", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := reuqest("/query", "1")
		r.Header.Add("ginError", "withMeta")
		srv.Router().ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		var rt graphql.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rt), w.Body.String())
		assert.Len(t, rt.Errors, 1)
		assert.EqualValues(t, 50000, rt.Errors[0].Extensions["code"])
		assert.EqualValues(t, "gin error {{field}}", rt.Errors[0].Message)
		assert.EqualValues(t, map[string]any{
			"field": "value",
		}, rt.Errors[0].Extensions["meta"])
	})
}
