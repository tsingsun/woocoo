package etcd3

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/testdata"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"log"
	"net"
	"testing"
	"time"
)

var cnf = testdata.Config

func TestRegistry_Apply(t *testing.T) {
	b := []byte(`
service:
  server:
    addr: :20000
    location: /woocoo/service
    version: "1.0"
  registry:
    schema: etcd
    ttl: 600
    etcd:
      endpoints:
        - 127.0.0.1:2379
      tls:
        ssl_certificate: ""
        ssl_certificate_key: ""
      dial-timeout: 3000
      dial-keep-alive-time: 3000
`)
	p, err := conf.NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	cfg := cnf.CutFromParser(p)
	r := New()
	if err != nil {
		t.Fatal(err)
	}
	r.Apply(cfg, "service.registry")
	if len(r.config.EtcdConfig.Endpoints) == 0 {
		t.Error("apply error")
	}
}

func TestRegisterResolver(t *testing.T) {
	listen := "localhost:9999"
	sn := "/group/test"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 10 * time.Second,
	}
	service := &registry.NodeInfo{
		ID:              "node1",
		ServiceLocation: sn,
		ServiceVersion:  "1.0",
		Address:         listen,
		Metadata:        nil,
	}
	reg, err := BuildFromConfig(&Config{
		EtcdConfig: etcdConfg,
		TTL:        10 * time.Minute,
	})

	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(service); err != nil {
		t.Fatalf("etcd register err: %s", err)
	}
	go func() {
		l, err := net.Listen("tcp", listen)
		if err != nil {
			t.Fatal("listen error:", err)
		}
		//srv := grpcx.Default()
		//helloworld.RegisterGreeterServer(srv.Engine(),&helloworld.Server{})
		//srv.Run()
		srv := grpc.NewServer()
		helloworld.RegisterGreeterServer(srv, &helloworld.Server{})
		if err := srv.Serve(l); err != nil {
			t.Fatal(err)
		}
	}()
	RegisterResolver(etcdConfg, sn)
	c, err := grpc.Dial("etcd:///", grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer c.Close()
	client := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {

		resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		time.Sleep(time.Second)
		log.Printf(resp.Message)
	}
}

func TestRegisterResolver2(t *testing.T) {
	//gn:="group"
	sn := "/woocoo/service"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 10 * time.Second,
	}
	go func() {
		srv := grpcx.New(grpcx.Configuration(cnf, "service"), grpcx.UseLogger())
		srv.Apply(cnf, "service")
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		if err := srv.Run(); err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(time.Second)
	RegisterResolver(etcdConfg, sn)
	c, err := grpc.Dial("etcd:///", grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer c.Close()
	client := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.SayHello(ctx, &helloworld.HelloRequest{Name: "round robin"})
		if err != nil {
			t.Error(err)
		} else {
			t.Log(resp.Message)
		}
	}
}
