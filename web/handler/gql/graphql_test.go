package gql

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/go-playground/assert/v2"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/vektah/gqlparser/v2/ast"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TODO
func TestDefaultGraphqlServer(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          handleFuncs:
            - graphql:
                queryPath: "/query"
                docPath: "/"
                group: "/graphql"
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.Configuration(cfg.Sub("web")), web.RegisterHandler("graphql", New()))
	gqlsrv := NewGraphqlServer(srv, &graphql.ExecutableSchemaMock{
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]interface{}) (int, bool) {
			panic("mock out the Complexity method")
		},
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			panic("mock out the Exec method")
		},
		SchemaFunc: func() *ast.Schema {
			//panic("mock out the Schema method")
			return nil
		},
	}, nil)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	gqlsrv.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
