package web

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/testco/wctest"
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
    addr: 127.0.0.1:33333
    tls:
      cert: "x509/server.crt"
      key: "x509/server.key"
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
    addr: 127.0.0.1:33333
    tls:
      cert: "x509/server.crt"
      key: "x509/server.key"
`
	cnf := conf.NewFromBytes([]byte(cfgStr)).AsGlobal()
	cnf.SetBaseDir(testdata.BaseDir())
	cnfWithouttls := cnf.Copy()
	cnfWithouttls.ParserOperator().Delete("web.server.tls")
	type fields struct {
		srv *Server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool // port conflict
	}{
		{name: "run-tls", fields: fields{New(WithConfiguration(cnf.Sub("web")))}},
		{name: "runGraceful-tls", fields: fields{New(WithConfiguration(cnf.Sub("web")), WithGracefulStop())}},
		{name: "runConflictPort", fields: fields{New(WithConfiguration(cnfWithouttls.Sub("web")))}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := tt.fields.srv
			wantErr := tt.wantErr
			err := wctest.RunWait(t, time.Millisecond*200, func() error {
				return srv.Run()
			}, func() error {
				if !wantErr {
					time.Sleep(time.Millisecond * 100)
					return srv.Stop(context.Background())
				}
				return nil
			})
			assert.NoError(t, err)
			if wantErr {
				srv1 := New(WithConfiguration(cnf.Sub("web")))
				assert.Error(t, srv1.Run())
				return
			}
		})
	}
}
