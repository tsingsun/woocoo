package etcd3

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
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
	"sync"
	"testing"
	"time"
)

var cnf = conf.New(conf.LocalPath(testdata.TestConfigFile()), conf.BaseDir(testdata.BaseDir())).Load()

func TestRegistry_Apply(t *testing.T) {
	b := []byte(`
service:
  server:
    addr: :20000
    location: /woocoo/service
    version: "1.0"
  registry:
    schema: etcd
    ttl: 600s
    etcd:
      endpoints:
        - 127.0.0.1:2379
      tls:
        ssl_certificate: ""
        ssl_certificate_key: ""
      dial-timeout: 3s
      dial-keep-alive-time: 3000
`)
	cfg := conf.NewFromBytes(b)
	r := New()
	r.Apply(cfg.Sub("service.registry"))
	if len(r.opts.EtcdConfig.Endpoints) == 0 {
		t.Error("apply error")
	}
}

func TestRegisterResolver(t *testing.T) {
	sn := "/group/test"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 10 * time.Second,
	}
	reg, err := BuildFromConfig(&Options{
		EtcdConfig: etcdConfg,
		TTL:        10 * time.Minute,
	})
	assert.NoError(t, err)
	reg.client.Delete(context.Background(), sn, clientv3.WithPrefix())
	var wg sync.WaitGroup
	wg.Add(2)
	runfunc := func(id, listen string) {
		service := &registry.NodeInfo{
			ID:              id,
			ServiceLocation: sn,
			ServiceVersion:  "1.0",
			Address:         listen,
			Metadata:        nil,
		}
		err = reg.Register(service)
		assert.NoError(t, err)
		l, err := net.Listen("tcp", listen)
		assert.NoError(t, err)
		srv := grpc.NewServer()
		helloworld.RegisterGreeterServer(srv, &helloworld.Server{})
		wg.Done()
		assert.NoError(t, srv.Serve(l))
	}

	go func() {
		runfunc("1", "localhost:9999")
	}()
	go func() {
		runfunc("2", "localhost:9998")
	}()
	wg.Wait()
	res, err := reg.client.Get(context.Background(), sn, clientv3.WithPrefix())
	assert.NoError(t, err)
	assert.EqualValues(t, res.Count, 2)
	RegisterResolver(etcdConfg, sn)
	c, err := grpc.Dial("etcd:///", grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	assert.NoError(t, err)
	defer c.Close()
	client := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {
		resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
	}
}

func TestRegistryGrpcx(t *testing.T) {
	//gn:="group"
	sn := "/woocoo/registrytest/1.0"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 10 * time.Second,
	}
	etcdCli, err := clientv3.New(etcdConfg)
	etcdCli.Delete(context.Background(), sn, clientv3.WithPrefix())

	cfg := cnf.Sub("grpc")
	cfg.Parser().Set("server.location", sn)
	srv := grpcx.New(grpcx.WithConfiguration(cfg), grpcx.UseLogger())
	go func() {
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		if err := srv.Run(); err != nil {
			t.Error(err)
		}
	}()
	cfg2 := cnf.Sub("grpc")
	cfg2.Parser().Set("server.location", sn)
	cfg2.Parser().Set("server.addr", ":20003")
	srv2 := grpcx.New(grpcx.WithConfiguration(cfg2), grpcx.UseLogger())
	go func() {
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		if err := srv2.Run(); err != nil {
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
		resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
	}
	{
		_, err = etcdCli.Delete(context.Background(), srv.NodeInfo.BuildKey())
		assert.NoError(t, err)
		time.Sleep(time.Millisecond * 100)
		_, err = client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		//todo robin validate
	}
	{
		srv2.Stop()
		//sleep let unregistry work,the latency 500 work fine, below will failure
		time.Sleep(time.Millisecond * 500)
		_, err = client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.Error(t, err)
	}

}
