package gql

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/vektah/gqlparser/v2/ast"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TODO
func TestRegistrySchema(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - graphql:
                queryPath: "/query"
                docPath: "/"
                group: "/"
      - graphql:
          basePath: "/graphql"
          middlewares:
            - graphql:
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.Configuration(cfg.Sub("web")), web.RegisterMiddleware(New()))
	gqlsrvList, err := RegisterSchema(srv, &graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]interface{}) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			panic("mock out the Exec method")
		},
		SchemaFunc: func() *ast.Schema {
			// panic("mock out the Schema method")
			return nil
		},
	}, &graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]interface{}) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			panic("mock out the Exec method")
		},
		SchemaFunc: func() *ast.Schema {
			// panic("mock out the Schema method")
			return nil
		},
	})
	assert.NoError(t, err)
	if !assert.Len(t, gqlsrvList, 2) {
		return
	}
	//g1 := gqlsrvList[0]
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	srv.Router().ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	g2 := gqlsrvList[0]
	r = httptest.NewRequest("GET", "/graphql", nil)
	w = httptest.NewRecorder()
	g2.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
