package authz

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/authz"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web/handler"
)

var cnf = `
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
    %s

handler:
  appCode: "test"
  ConfigPath: "authz"
`

func TestAuthorizer(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	tcnf := fmt.Sprintf(cnf, "m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act")
	authz.SetAdapter(stringadapter.NewAdapter(`p, 1, /, GET`))
	tests := []struct {
		name  string
		cfg   *conf.Configuration
		req   *http.Request
		check func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "pass",
			cfg:  conf.NewFromBytes([]byte(tcnf)).Sub("handler"),
			req: httptest.NewRequest("GET", "/", nil).
				WithContext(security.WithContext(context.Background(),
					security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"}))),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "no pass",
			cfg:  conf.NewFromBytes([]byte(tcnf)).Sub("handler"),
			req: httptest.NewRequest("GET", "/unauth", nil).
				WithContext(security.WithContext(context.Background(), security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"}))),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			name: "match error",
			cfg: func() *conf.Configuration {
				nm := fmt.Sprintf(cnf, "m = g(r.sub, p.sub) && r.obj1 == p.obj && r.act != p.act")
				c := conf.NewFromBytes([]byte(nm)).Sub("handler")
				return c
			}(),
			req: httptest.NewRequest("GET", "/", nil).
				WithContext(security.WithContext(context.Background(),
					security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"}))),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, w.Code)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New()
			defer got.Shutdown(context.Background())
			h := handler.NewSimpleMiddleware("authz", got.ApplyFunc)
			w := httptest.NewRecorder()
			_, e := gin.CreateTestContext(w)
			e.ContextWithFallback = true
			e.Use(h.ApplyFunc(tt.cfg))
			e.GET("/", func(c *gin.Context) {
				c.String(200, "ok")
			})
			e.GET("/unauth", func(c *gin.Context) {
				c.String(200, "ok")
			})

			e.ServeHTTP(w, tt.req)
			tt.check(t, w)
		})
	}
}
