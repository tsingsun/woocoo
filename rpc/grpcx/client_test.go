package grpcx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/pkg/conf"
	_ "github.com/tsingsun/woocoo/rpc/grpcx/registry/etcd3"
	"github.com/tsingsun/woocoo/test/testdata"
	"google.golang.org/grpc/connectivity"
	"testing"
	"time"
)

func TestClient_DialRegistry(t *testing.T) {
	b := []byte(`
service:
  server:
    addr: 127.0.0.1:20005
    namespace: woocoo
    version: "1.0"
    engine:
      - keepalive:
          time: 3600s
  registry:
    scheme: etcd
    ttl: 600s
    etcd:
      tls:
        cert: "x509/test.pem"
        key: "x509/test.key"
      endpoints:
        - 127.0.0.1:2379
      dial-timeout: 3s
      dial-keep-alive-time: 1h
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata: 
        version: "1.0"
    grpcDialOption:
      - insecure:
      - block:
      - timeout: 5s
      - tls:
          cert: "x509/test.pem" 
      - unaryInterceptors:
          - otel:
`)
	cfg := conf.NewFromBytes(b, conf.WithBaseDir(testdata.BaseDir()))
	srv := New(WithConfiguration(cfg.Sub("service")))
	go func() {
		if err := srv.Run(); err != nil {
			t.Error(err)
			return
		}
	}()
	time.Sleep(2000)
	cli := NewClient(cfg.Sub("service"))
	assert.Equal(t, cli.dialOpts.Namespace, "woocoo")
	assert.Equal(t, cli.dialOpts.ServiceName, "helloworld.Greeter")
	assert.EqualValues(t, cli.dialOpts.Metadata, map[string]string{"version": "1.0"})
	_, err := cli.Dial("")
	assert.Error(t, err)
	srv.Stop(context.Background())
}

func TestClient_DialNaming(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20001
    namespace: woocoo
    version: "1.0"
    engine:
      - unaryInterceptors:
          - trace:
          - accessLog:
              timestampFormat: "2006-01-02 15:04:05"
          - recovery:
      - streamInterceptors:
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      version: "1.0"
      metadata:  
        dst_location: amoy
        src_tag: tag1
        headerPrefix: "head1,head2"
    grpcDialOption:
      - tls:
      - block:
      - timeout: 5s
      - defaultServiceConfig: '{ "loadBalancingConfig": [{"round_robin": {}}] }'
      - unaryInterceptors:
          - otel:
`)
	cfg := conf.NewFromBytes(b)
	go func() {
		srv := New(WithConfiguration(cfg.Sub("grpc")))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		if err := srv.Run(); err != nil {
			t.Error(err)
			return
		}
	}()
	time.Sleep(time.Second)
	cli := NewClient(cfg.Sub("grpc"))
	conn, err := cli.Dial(cfg.String("grpc.server.addr"))
	assert.NoError(t, err)
	if conn.GetState() != connectivity.Ready {
		t.Fail()
	}
}
