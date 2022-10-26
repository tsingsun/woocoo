package web

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	srv := New(WithGracefulStop())
	assert.Equal(t, srv.ServerOptions().Addr, defaultAddr)
	assert.NotNil(t, srv.HandlerManager())
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
		srv        *Server
		wantStatus int
	}{
		{
			name: "normal",
			srv: New(WithConfiguration(cfg.Sub("web")),
				RegisterMiddlewareByFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
					return func(c *gin.Context) {
						c.Next()
					}
				})),
			wantStatus: 200,
		},
		{
			name: "registerHandlerAbort",
			srv: New(WithConfiguration(cfg.Sub("web")),
				RegisterMiddlewareByFunc("test", func(cfg *conf.Configuration) gin.HandlerFunc {
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
		srv *Server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool // port conflict
	}{
		{name: "run", fields: fields{New(WithConfiguration(cnf.Sub("web")))}},
		{name: "runGraceful", fields: fields{New(WithConfiguration(cnf.Sub("web")), WithGracefulStop())}},
		{name: "runConflictPort", fields: fields{New(WithConfiguration(cnf.Sub("web")))}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wctest.RunWait(t, time.Second*2, func() error {
				return tt.fields.srv.Run()
			}, func() error {
				if !tt.wantErr {
					time.Sleep(time.Second)
					return tt.fields.srv.Stop(context.Background())
				}
				return nil
			})
			assert.NoError(t, err)
			if tt.wantErr {
				srv := New(WithConfiguration(cnf.Sub("web")))
				assert.Error(t, srv.Run())
				return
			}
		})
	}
}
