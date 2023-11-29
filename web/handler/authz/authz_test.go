package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web/handler"
)

type mockAuthorizer struct {
	// user-role map
	users map[string]string
}

func (m mockAuthorizer) Conv(kind security.ArnRequestKind, arnParts ...string) security.Resource {
	switch kind {
	case security.ArnRequestKindWeb:
		return security.Resource(strings.Join(append(arnParts[:1], arnParts[2:]...), security.ArnSplit))
	default:

	}
	return security.Resource(strings.Join(arnParts, security.ArnSplit))
}

func (m mockAuthorizer) Eval(ctx context.Context, identity security.Identity, item security.Resource) (bool, error) {
	if identity.Name() == "2" {
		return false, nil
	}
	return item.MatchResource("test:/"), nil
}

func (m mockAuthorizer) QueryAllowedResourceConditions(ctx context.Context, identity security.Identity, item security.Resource) ([]string, error) {
	panic("not used in this test")
}

func TestAuthorizer(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	security.SetDefaultAuthorizer(&mockAuthorizer{})

	var cnf = `
handler:
  appCode: "test"
`
	tests := []struct {
		name  string
		cfg   *conf.Configuration
		req   *http.Request
		check func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "pass",
			cfg:  conf.NewFromBytes([]byte(cnf)).Sub("handler"),
			req: httptest.NewRequest("GET", "/", nil).
				WithContext(security.WithContext(context.Background(),
					security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"}))),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "no pass",
			cfg:  conf.NewFromBytes([]byte(cnf)).Sub("handler"),
			req: httptest.NewRequest("GET", "/unauth", nil).
				WithContext(security.WithContext(context.Background(),
					security.NewGenericPrincipalByClaims(map[string]any{"sub": "1"}))),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			name: "match error",
			cfg: func() *conf.Configuration {
				c := conf.NewFromBytes([]byte(cnf)).Sub("handler")
				return c
			}(),
			req: httptest.NewRequest("GET", "/", nil).
				WithContext(security.WithContext(context.Background(),
					security.NewGenericPrincipalByClaims(map[string]any{"sub": "2"}))),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			name: "miss user",
			cfg: func() *conf.Configuration {
				c := conf.NewFromBytes([]byte(cnf)).Sub("handler")
				return c
			}(),
			req: httptest.NewRequest("GET", "/", nil),
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, w.Code)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Middleware()
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
