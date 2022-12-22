package woocoo

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/web"
	"testing"
	"time"

	_ "github.com/tsingsun/woocoo/rpc/grpcx/registry/etcd3"
)

func TestApp(t *testing.T) {
	cnf := wctest.Configuration()
	app := New(WithAppConfiguration(cnf))
	websrv := web.New(web.WithConfiguration(app.AppConfiguration().Sub("web")))
	grpcsrv := grpcx.New(grpcx.WithConfiguration(app.AppConfiguration().Sub("grpc")), grpcx.WithGrpcLogger())
	time.AfterFunc(time.Second*2, func() {
		t.Log("stop")
		app.Stop()
	})
	app.RegisterServer(websrv, grpcsrv)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestSampleWeb(t *testing.T) {
	cnf := conf.New()
	app := New(WithAppConfiguration(cnf))
	websrv := web.New()
	websrv.Router().GET("/", func(c *gin.Context) {
		c.String(200, "hello world")
	})
	time.AfterFunc(time.Second, func() {
		app.Stop()
	})
	app.RegisterServer(websrv)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
}
