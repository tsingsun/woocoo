package grpcx

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
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
      sslCertificate: "x509/test.pem"
      sslCertificateKey: "x509/test.key"
  - unaryInterceptors:
    - accessLog:
        timestampFormat: "2006-01-02 15:04:05"
    - recovery:
`)
	// testdata.Path("x509/test.pem"), testdata.Path("x509/test.key")
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
	cfg.Load()
	s := New(WithConfiguration(cfg.Sub("grpc")),
		WithGracefulStop(),
		WithGrpcOption(grpc.ConnectionTimeout(1000)),
		UseLogger(),
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
			tt.wantErr(t, s.Run(), fmt.Sprintf("Run()"))
		})
	}
}

func TestServer_UseLogger(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20000
  engine:
  - unaryInterceptors:
    - accessLog:  
`)
	// testdata.Path("x509/test.pem"), testdata.Path("x509/test.key")
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
	cfg.Load()
	s := New(WithConfiguration(cfg.Sub("grpc")),
		UseLogger(),
	)
	assert.IsType(t, log.Component(log.GrpcComponentName).Logger().ContextLogger(), &interceptor.GrpcContextLogger{})
	assert.IsType(t, log.Component(interceptor.AccessLogComponentName).Logger().ContextLogger(), &interceptor.GrpcContextLogger{})
	assert.NotNil(t, s)
}
