package client_test

import (
	"bytes"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/client"
	_ "github.com/tsingsun/woocoo/rpc/grpcx/registry/etcd3"
	"github.com/tsingsun/woocoo/test/testdata"
	"google.golang.org/grpc/connectivity"
	"testing"
	"time"
)

var (
	cnf = testdata.Config
)

func TestClient_Dial(t *testing.T) {
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
      - streamInterceptors:
  registry:
    schema: etcd
    ttl: 600
    etcd:
      tls:
        ssl_certificate: ""
        ssl_certificate_key: ""
      endpoints:
        - localhost:2379
      dial-timeout: 3
      dial-keep-alive-time: 3
  prometheus:
    addr: 0.0.0.0:2222
  client:
    - insecure:
    - block:
    - timeout: 5
    - unaryInterceptors:
        - trace:
`)
	p, err := conf.NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	cfg := cnf.CutFromParser(p)
	go func() {
		srv := grpcx.New(grpcx.Configuration(cfg, "service"))
		if err := srv.Run(); err != nil {
			t.Fatal(err)
		}
	}()
	time.Sleep(1000)
	cli := client.New(client.Configuration(cfg, "service"))
	if err != nil {
		t.Error(err)
	}
	conn, err := cli.Dial()
	if err != nil {
		t.Fatal(err)
	}
	if conn.GetState() != connectivity.Ready {
		t.Fail()
	}
}
