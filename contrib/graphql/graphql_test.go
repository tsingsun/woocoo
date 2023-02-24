package gql

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
      - graphql:
          basePath: "/graphql"
          middlewares:
            - graphql:
                queryPath: "/query"
                docPath: "/doc"
                group: "/graphql"
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.WithConfiguration(cfg.Sub("web")), web.RegisterMiddleware(New()))
	gqlsrvList, err := RegisterSchema(srv, &graphql.ExecutableSchemaMock{
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
	}, &graphql.ExecutableSchemaMock{
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
	})
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
	})
	t.Run("test sub", func(t *testing.T) {
		g2 := gqlsrvList[0]
		r := httptest.NewRequest("GET", "/graphql", nil)
		w := httptest.NewRecorder()
		g2.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}
