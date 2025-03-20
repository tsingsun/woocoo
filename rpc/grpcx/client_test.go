package grpcx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/mock/helloworld"
	mock "github.com/tsingsun/woocoo/test/mock/registry"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/wctest"
	"google.golang.org/grpc/connectivity"
	"testing"
	"time"

	_ "github.com/tsingsun/woocoo/test/mock/registry"
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
      - connectParams:
          backoff:
            maxDelay: 10s
          minConnectTimeout: 5s
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
				assert.Equal(t, client.withTransportCredentials, true)
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
				assert.Equal(t, client.withTransportCredentials, true)
				assert.Len(t, client.dialOptions, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.args.cfg)
			if tt.panic {
				assert.Error(t, err)
				return
			}
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
	t.Run("dial target", func(t *testing.T) {
		cli, err := NewClient(cfg.Sub("grpc"))
		require.NoError(t, err)
		assert.True(t, cli.withTransportCredentials)
		conn, err := cli.Dial(cfg.String("grpc.server.addr"))
		assert.NoError(t, err)
		// timeout is not a dial option
		assert.Len(t, cli.dialOptions, 5)
		if conn.GetState() != connectivity.Ready {
			t.Fail()
		}
		gc := helloworld.NewGreeterClient(conn)
		resp, err := gc.SayHello(context.Background(), &helloworld.HelloRequest{Name: "woocoo"})
		require.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello woocoo")
	})
	t.Run("dial empty", func(t *testing.T) {
		cli, err := NewClient(cfg.Sub("grpc"))
		require.NoError(t, err)
		assert.True(t, cli.withTransportCredentials)
		conn, err := cli.Dial("")
		assert.NoError(t, err)
		// timeout is not a dial option
		assert.Len(t, cli.dialOptions, 5)
		if conn.GetState() != connectivity.Ready {
			t.Fail()
		}
	})
}

func TestClient_DialRegistry(t *testing.T) {
	mock.RegisterDriver(map[string]*registry.ServiceInfo{
		"helloworld.Greeter": {
			Name:    "helloworld.Greeter",
			Version: "1.0",
			Host:    "localhost",
			Port:    20005,
		},
	})
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
    scheme: mock
    ttl: 600s 
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata: 
        version: "1.0"
    dialOption:
      - insecure:
      - block:
      - unaryInterceptors:
          - otel:
`)
	cfg := conf.NewFromBytes(b, conf.WithBaseDir(testdata.BaseDir()))
	srv := New(WithConfiguration(cfg.Sub("service")))
	helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
	require.NoError(t, wctest.RunWait(t.Log, time.Second, func() error {
		return srv.Run()
	}))
	defer srv.Stop(context.Background())
	cli, err := NewClient(cfg.Sub("service"))
	require.NoError(t, err)
	assert.Equal(t, cli.registryOptions.Namespace, "woocoo")
	assert.Equal(t, cli.registryOptions.ServiceName, "helloworld.Greeter")
	assert.EqualValues(t, cli.registryOptions.Metadata, map[string]string{"version": "1.0"})
	_, err = cli.Dial("")
	assert.NoError(t, err)

	t.Run("downgrade", func(t *testing.T) {
		_, err := cli.Dial(cfg.String("service.server.addr"))
		require.NoError(t, err)
	})
}
