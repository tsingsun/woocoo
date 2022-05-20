package web_test

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	_ "github.com/tsingsun/woocoo/web/handler/gql"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

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
    sslCertificate: ""
    sslCertificateKey: ""
  engine:
    redirectTrailingSlash: false
    remoteIPHeaders:
      - X-Forwarded-For
      - X-Real-XIP
    routerGroups:
      - default:
          middlewares: 
            - accessLog:
                exclude:
                  - IntrospectionQuery
            - recovery:
            - test:
      - auth:
          basePath: "/auth"
          middlewares:
            - jwt: 
                signingKey: 12345678
                privKey: config/privKey.pem
                pubKey: config/pubKey.pem  
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	tests := []struct {
		name       string
		srv        *web.Server
		wantStatus int
	}{
		{
			name: "normal",
			srv: web.New(web.Configuration(cfg.Sub("web")), web.RegisterMiddlewareByFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
				return func(c *gin.Context) {
					c.Next()
				}
			})),
			wantStatus: 200,
		},
		{
			name: "registerHandlerAbort",
			srv: web.New(web.Configuration(cfg.Sub("web")), web.RegisterMiddlewareByFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
				return func(c *gin.Context) {
					c.AbortWithStatus(500)
				}
			})),
			wantStatus: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/user/123", nil)
			w := httptest.NewRecorder()
			tt.srv.Router().Engine.GET("/user/:id", func(c *gin.Context) {
				c.String(200, "User")
			})
			tt.srv.Router().Engine.ServeHTTP(w, r)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestServer_Run(t *testing.T) {
	cfgStr := `
web:
  server:
    addr: 0.0.0.0:33333
    sslCertificate: ""
    sslCertificateKey: ""
`
	cnf := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	type fields struct {
		srv *web.Server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool //port conflict
	}{
		{name: "run", fields: fields{web.New(web.Configuration(cnf.Sub("web")))}},
		{name: "runGracefull", fields: fields{web.New(web.Configuration(cnf.Sub("web")), web.GracefulStop())}},
		{name: "runConflictPort", fields: fields{web.New(web.Configuration(cnf.Sub("web")))}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			if tt.wantErr {
				go func() {
					srv1 := web.New(web.Configuration(cnf.Sub("web")))
					srv1.Run()
				}()
				time.Sleep(time.Second)
			}
			go func() {
				defer wg.Done()
				err := tt.fields.srv.Run()
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}()
			go func() {
				time.Sleep(time.Second * 1)
				tt.fields.srv.Stop()
			}()
			wg.Wait()
		})
	}
}
