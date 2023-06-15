package authz

import (
	"context"
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

func TestNewAuthorizer(t *testing.T) {
	authz.SetAdapter(stringadapter.NewAdapter(`p, 1, /, GET`))
	tests := []struct {
		name string
		cfg  *conf.Configuration
	}{
		{
			name: "NewAuthorizer",
			cfg: conf.NewFromBytes([]byte(`
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

handler:
  appCode: "test"
  ConfigPath: "authz"
`)).Sub("handler"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New()
			h := handler.NewSimpleMiddleware("authz", got.ApplyFunc)
			req := httptest.NewRequest("GET", "/", nil).
				WithContext(security.WithContext(context.Background(), security.NewGenericPrincipalByClaims(map[string]interface{}{"sub": "1"})))
			w := httptest.NewRecorder()
			_, e := gin.CreateTestContext(w)
			e.ContextWithFallback = true
			e.Use(h.ApplyFunc(tt.cfg))
			e.ServeHTTP(w, req)
			assert.Equal(t, 404, w.Code) // no found router is right
			assert.NoError(t, got.Shutdown(context.Background()))
		})
	}
}
