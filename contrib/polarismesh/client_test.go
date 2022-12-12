package polarismesh

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/client"
	"google.golang.org/grpc/grpclog"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func init() {
	log := grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(log)
}

func TestClient_Dial(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 0.0.0.0:20011
    namespace: woocoo
    version: "1.0"
    engine:
      - unaryInterceptors:
          - trace:
          - accessLog:
              timestampFormat: "2006-01-02 15:04:05"
          - recovery:
      - streamInterceptors:
  registry:
    scheme: polaris
    ttl: 600s
    polaris: 
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata: 
        version: "1.0"
        dst_location: amoy
        src_tag: tag1
        headerPrefix: "head1,head2"
    grpcDialOption:
      - insecure:
      - block:
      - timeout: 5s
      - defaultServiceConfig: '{ "loadBalancingConfig": [{"polaris": {}}] }'
      - unaryInterceptors:
          - trace:
`)
	cfg := conf.NewFromBytes(b)
	err := wctest.RunWait(t, time.Second*2, func() error {
		srv := grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	})
	require.NoError(t, err)
	cli := client.New(cfg.Sub("grpc"))
	conn, err := cli.Dial("")
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	gclient := helloworld.NewGreeterClient(conn)
	resp, err := gclient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "world"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_DialMultiServer(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 0.0.0.0:20012
    namespace: woocoo
    version: "1.0"
    engine:
      - unaryInterceptors: 
          - accessLog:
              timestampFormat: "2006-01-02 15:04:05"
          - recovery:
      - streamInterceptors:
  registry:
    scheme: polaris
    ttl: 600s
    polaris: 
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata: 
        version: "1.0"
        dst_location: amoy
        src_tag: tag1
        headerPrefix: "head1,head2"
    grpcDialOption:
      - insecure:
      - block:
      - timeout: 30s
      - defaultServiceConfig: '{ "loadBalancingConfig": [{"polaris": {}}] }' 
`)
	cfg := conf.NewFromBytes(b)
	var srv, srv2 *grpcx.Server
	err := wctest.RunWait(t, time.Second*5, func() error {
		srv = grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")), grpcx.UseLogger())
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	}, func() error {
		cfg2 := cfg.Sub("grpc")
		cfg2.Parser().Set("server.addr", ":20013")
		srv2 = grpcx.New(grpcx.WithConfiguration(cfg2), grpcx.UseLogger())
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		return srv2.Run()
	})
	require.NoError(t, err)
	cli := client.New(cfg.Sub("grpc"))
	c, err := cli.Dial("")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, c)
	defer c.Close()
	hcli := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {
		resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
	}
	{
		srv.Stop(context.Background())
		time.Sleep(time.Second * 2)
		resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
	}
	{
		srv2.Stop(context.Background())
		//sleep let unregistry work,the latency 500 work fine, below will failure
		resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.Nil(t, resp)
		assert.Error(t, err)
		//time.Sleep(time.Second)
	}
}
