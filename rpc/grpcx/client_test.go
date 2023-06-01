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

func TestClient_New(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20001
    namespace: woocoo
    version: "1.0"
    engine:
      - compression:
          name: gzip
          level: 1
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
    dialOption:
      - tls:
      - compression:
          name: gzip
      - block:
      - timeout: 5s
      - serviceConfig: '{ "loadBalancingConfig": [{"round_robin": {}}] }'
      - unaryInterceptors:
          - otel:
`)
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name  string
		args  args
		panic bool
		check func(*Client)
	}{
		{
			name: "all",
			args: args{
				cfg: conf.NewFromBytes(b).Sub("grpc"),
			},
			check: func(c *Client) {

			},
		},
		{
			name: "empty",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{}),
			},
			check: func(client *Client) {
				assert.Equal(t, client.withSecure, true)
				assert.Len(t, client.dialOptions, 1)
			},
		},
		{
			name: "tls",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"client": map[string]any{
						"diaOption": []any{
							map[string]any{
								"tls": nil,
							},
						},
					},
				}),
			},
			check: func(client *Client) {
				assert.Equal(t, client.withSecure, true)
				assert.Len(t, client.dialOptions, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				assert.Panics(t, func() {
					NewClient(tt.args.cfg)
				})
				return
			}
			got := NewClient(tt.args.cfg)
			assert.NotNil(t, got)
		})
	}
}

func TestClient_NewAndDialNaming(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20001
    namespace: woocoo
    version: "1.0"
    engine:
      - compression:
          name: gzip
          level: 1
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
    dialOption:
      - tls:
      - compression:
          name: gzip
      - block:
      - timeout: 5s
      - serviceConfig: '{ "loadBalancingConfig": [{"round_robin": {}}] }'
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
	assert.True(t, cli.withSecure)
	conn, err := cli.Dial(cfg.String("grpc.server.addr"))
	assert.NoError(t, err)
	// timeout is not a dial option
	assert.Len(t, cli.dialOptions, 5)
	if conn.GetState() != connectivity.Ready {
		t.Fail()
	}
	gc := helloworld.NewGreeterClient(conn)
	resp, err := gc.SayHello(context.Background(), &helloworld.HelloRequest{Name: "woocoo"})
	assert.NoError(t, err)
	assert.Equal(t, resp.Message, "Hello woocoo")
}

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
        cert: "x509/server.crt"
        key: "x509/server.key"
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
    dialOption:
      - insecure:
      - block:
      - timeout: 5s
      - tls:
          cert: "x509/server.crt" 
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
	assert.Equal(t, cli.registryOptions.Namespace, "woocoo")
	assert.Equal(t, cli.registryOptions.ServiceName, "helloworld.Greeter")
	assert.EqualValues(t, cli.registryOptions.Metadata, map[string]string{"version": "1.0"})
	_, err := cli.Dial("")
	assert.Error(t, err)
	srv.Stop(context.Background())
}
