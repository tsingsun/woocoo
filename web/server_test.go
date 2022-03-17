package web_test

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	_ "github.com/tsingsun/woocoo/web/handler/gql"
	"net/http/httptest"
	"testing"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestNew(t *testing.T) {
	srv := web.New()
	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	srv.Router().Engine.GET("/user/:id", func(c *gin.Context) {
		c.String(200, "User")
	})
	srv.Router().Engine.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
}

func TestServer_Apply(t *testing.T) {
	cfgStr := `
web:
  server:
    addr: 0.0.0.0:33333
    ssl_certificate: ""
    ssl_certificate_key: ""
  engine:
    redirectTrailingSlash: false
    remoteIPHeaders:
      - X-Forwarded-For
      - X-Real-XIP
    routerGroups:
      - default:
          handleFuncs: 
            - accessLog:
                exclude:
                  - IntrospectionQuery
            - recovery:
      - auth:
          basePath: "/auth"
          handleFuncs:
            - auth:
                realm: woocoo
                secret: 12345678
                privKey: config/privKey.pem
                pubKey: config/pubKey.pem
                tenantHeader: Qeelyn-Org-Id
                disabledExpiredCheck: false
  ##以下配置暂时用于 标识出哪些路由为被客制化的
  routerules:
    - path: /
      handler: "GraphQL DevTool"
      method: get
    - path: /query
      handler: "GraphQL Query"
      method: post
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	srv := web.NewBuiltIn(web.Configuration(cfg))
	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	srv.Router().GET("/user/:id", func(c *gin.Context) {
		c.String(200, "User")
	})
	srv.Router().ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
}
