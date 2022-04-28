package client

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	_ "github.com/tsingsun/woocoo/rpc/grpcx/registry/etcd3"
	"google.golang.org/grpc/connectivity"
	"testing"
	"time"
)

func TestClient_DialRegistry(t *testing.T) {
	b := []byte(`
service:
  server:
    addr: :20005
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
        sslCertificate: ""
        sslCertificateKey: ""
      endpoints:
        - localhost:12379
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
      - unaryInterceptors:
          - trace:
`)
	cfg := conf.NewFromBytes(b)
	srv := grpcx.New(grpcx.WithConfiguration(cfg.Sub("service")))
	go func() {
		if err := srv.Run(); err != nil {
			t.Error(err)
			return
		}
	}()
	time.Sleep(2000)
	cli := New(Configuration(cfg.Sub("service")))
	assert.Equal(t, cli.dialOpts.Namespace, "woocoo")
	assert.Equal(t, cli.dialOpts.ServiceName, "helloworld.Greeter")
	assert.EqualValues(t, cli.dialOpts.Metadata, map[string]string{"version": "1.0"})
	_, err := cli.Dial("")
	assert.Error(t, err)
	srv.Stop()
}

func TestClient_DialNaming(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: :20001
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
      metadata: 
        version: "1.0"
        dst_location: amoy
        src_tag: tag1
        headerPrefix: "head1,head2"
    grpcDialOption:
      - insecure:
      - block:
      - timeout: 5s
      - defaultServiceConfig: '{ "loadBalancingConfig": [{"round_robin": {}}] }'
      - unaryInterceptors:
          - trace:
`)
	cfg := conf.NewFromBytes(b)
	go func() {
		srv := grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		if err := srv.Run(); err != nil {
			t.Error(err)
			return
		}
	}()
	time.Sleep(time.Second)
	cli := New(Configuration(cfg.Sub("grpc")))
	conn, err := cli.Dial(cfg.String("grpc.server.addr"))
	assert.NoError(t, err)
	if conn.GetState() != connectivity.Ready {
		t.Fail()
	}
}
