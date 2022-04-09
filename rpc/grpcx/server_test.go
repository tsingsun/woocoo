package grpcx_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"testing"
)

func TestNew(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: :20000
    location: /woocoo/service
    version: "1.0"
    registryMeta:
      key1: value1
      key2: value2
  engine:
  - keepalive:
      time: 1h
  - tls:
      ssl_certificate: ""
      ssl_certificate_key: ""
  - unaryInterceptors:
    - trace:
    - accessLog:
        timestampFormat: "2006-01-02 15:04:05"
    - recovery:
    - prometheus:
    - auth:
        signingAlgorithm: HS256
        realm: woocoo
        secret: 123456
        privKey: config/privKey.pem
        pubKey: config/pubKey.pem
        tenantHeader: Qeelyn-Org-Id
`)
	cfg := conf.NewFromBytes(b).Load()
	s := grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
	assert.NotNil(t, s)
}
