package grpcx_test

import (
	"bytes"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf = testdata.Config
)

func TestServer_Apply(t *testing.T) {
	b := []byte(`
service:
  server:
    addr: :20000
    location: /woocoo/service
    version: "1.0"
  engine:
  - keepalive:
      time: 3600
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
	p, err := conf.NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	cfg := cnf.CutFromParser(p)
	s := grpcx.New()
	s.Apply(cfg, "service")
}
