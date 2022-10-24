package grpcx_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/test/testdata"
	"google.golang.org/grpc"
	"testing"
)

func TestNew(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: :20000
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
    - auth:
        signingAlgorithm: HS256
        realm: woocoo
        secret: 123456
        privKey: config/privKey.pem
        pubKey: config/pubKey.pem
        tenantHeader: Qeelyn-Org-Id
`)
	//testdata.Path("x509/test.pem"), testdata.Path("x509/test.key")
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
	cfg.Load()
	s := grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")),
		grpcx.GracefulStop(),
		grpcx.WithGrpcOption(grpc.ConnectionTimeout(1000)))
	assert.NotNil(t, s)
}
