package woocoo

import (
	"github.com/tsingsun/woocoo/internal/wctest"
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
	grpcsrv := grpcx.New(grpcx.WithConfiguration(app.AppConfiguration().Sub("grpc")))
	time.AfterFunc(time.Second*1, func() {
		app.Stop()
	})
	app.RegisterServer(websrv, grpcsrv)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
}
