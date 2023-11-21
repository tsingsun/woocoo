package grpcx

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/mock/helloworld"
	"github.com/tsingsun/woocoo/test/testdata"
	"google.golang.org/grpc"
	"testing"
	"time"
)

type MockRegistry struct {
}

// Register a service node
func (mr *MockRegistry) Register(serviceInfo *registry.ServiceInfo) error {
	if serviceInfo.Name == "error" {
		return errors.New("register error")
	}
	return nil
}

// Unregister a service node
func (mr *MockRegistry) Unregister(serviceInfo *registry.ServiceInfo) error {
	if serviceInfo.Name == "error" {
		return errors.New("unregister error")
	}
	return nil
}

// TTL returns the time to live of the service node, if it is not available, return 0.
// every tick will call Register function to refresh.
func (mr *MockRegistry) TTL() time.Duration {
	return time.Second
}
func (mr *MockRegistry) Close() {

}

// GetServiceInfos returns the members of the cluster by service name
func (mr *MockRegistry) GetServiceInfos(_ string) ([]*registry.ServiceInfo, error) {
	return nil, nil
}

func TestNew(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20000
    namespace: /woocoo/service
    version: "1.0"
    registryMeta:
      key1: value1
      key2: value2
  engine:
  - keepalive:
      time: 1h
  - tls:
      cert: "x509/server.crt"
      key: "x509/server.key"
  - unaryInterceptors:
    - accessLog:
        timestampFormat: "2006-01-02 15:04:05"
    - recovery:
  - streamInterceptors:
    - accessLog:
`)
	// testdata.Path("x509/test.pem"), testdata.Path("x509/test.key")
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
	cfg.Load()
	s := New(WithConfiguration(cfg.Sub("grpc")),
		WithGracefulStop(),
		WithGrpcOption(grpc.ConnectionTimeout(1000)),
	)
	assert.NotNil(t, s)
}

func TestServer_Run(t *testing.T) {
	type fields struct {
		opts         serverOptions
		engine       *grpc.Server
		exit         chan chan error
		registry     registry.Registry
		ServiceInfos []*registry.ServiceInfo
	}
	tests := []struct {
		name    string
		fields  fields
		service bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test",
			fields: fields{
				opts: serverOptions{
					Addr:         "127.0.0.1:11000",
					gracefulStop: true,
				},
				engine:   grpc.NewServer(),
				registry: &MockRegistry{},
				exit:     make(chan chan error),
			},
			service: true,
			wantErr: assert.NoError,
		},
		{
			name: "test registry error",
			fields: fields{
				opts: serverOptions{
					Addr: "127.0.0.1:11001",
				},
				engine:   grpc.NewServer(),
				registry: &MockRegistry{},
				exit:     make(chan chan error),
				ServiceInfos: []*registry.ServiceInfo{
					{
						Name: "error",
					},
				},
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				opts:         tt.fields.opts,
				engine:       tt.fields.engine,
				exit:         tt.fields.exit,
				registry:     tt.fields.registry,
				ServiceInfos: tt.fields.ServiceInfos,
			}
			if tt.service {
				helloworld.RegisterGreeterServer(s.Engine(), &helloworld.Server{})
			}
			time.AfterFunc(time.Second*2, func() {
				s.Stop(context.Background())
			})
			tt.wantErr(t, s.Run(), "Run()")
		})
	}
}

// TODO grpclog will case test fail by race condition.so let it skip
func serverUselogger(t *testing.T) { //nolint:unused
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20000
  engine:
  - unaryInterceptors:
    - accessLog:  
`)
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
	cfg.Load()
	s := New(WithConfiguration(cfg.Sub("grpc")),
		WithGrpcLogger(),
	)
	assert.IsType(t, log.Component(log.GrpcComponentName).Logger().ContextLogger(), &interceptor.GrpcContextLogger{})
	assert.IsType(t, log.Component(interceptor.AccessLogComponentName).Logger().ContextLogger(), &interceptor.GrpcContextLogger{})
	assert.NotNil(t, s)
}

func TestCustomRegisterGrpc(t *testing.T) {
	RegisterGrpcServerOption("sopt", func(configuration *conf.Configuration) grpc.ServerOption {
		return grpc.EmptyServerOption{}
	})
	RegisterGrpcUnaryInterceptor("uiopt", func(configuration *conf.Configuration) grpc.UnaryServerInterceptor {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			return nil, nil
		}
	})
	RegisterGrpcStreamInterceptor("siopt", func(configuration *conf.Configuration) grpc.StreamServerInterceptor {
		return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return nil
		}
	})
	RegisterUnaryClientInterceptor("uciopt", func(configuration *conf.Configuration) grpc.UnaryClientInterceptor {
		return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return nil
		}
	})
	RegisterStreamClientInterceptor("sciopt", func(configuration *conf.Configuration) grpc.StreamClientInterceptor {
		return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return nil, nil
		}
	})
	RegisterDialOption("dopt", func(configuration *conf.Configuration) grpc.DialOption {
		return grpc.EmptyDialOption{}
	})
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:0
  engine:
  - sopt: 
  - unaryInterceptors:
    - uiopt:
    - recovery:
  - streamInterceptors:
    - siopt:
  client:
    dialOption:
    - dopt:
    - block:
    - unaryInterceptors:
      - uciopt:
    - streamInterceptors:
      - sciopt:
`)
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
	cfg.Load()
	s := New(WithConfiguration(cfg.Sub("grpc")))
	assert.NotNil(t, s)
}
