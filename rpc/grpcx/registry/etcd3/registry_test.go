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
	"github.com/tsingsun/woocoo/test/testproto"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

var testcnf = conf.New(conf.LocalPath(testdata.TestConfigFile()), conf.BaseDir(testdata.BaseDir())).Load()

func TestRegistry_Apply(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: :20000
    nameSpace: /woocoo/service
    version: "1.0"
    ipv6: true
  registry:
    scheme: etcd
    ttl: 600s
    etcd:
      endpoints:
        - 127.0.0.1:2379
      tls:
        sslCertificate: ""
        sslCertificateKey: ""
      dial-timeout: 3s
      dial-keep-alive-time: 1h
`)
	cfg := conf.NewFromBytes(b)
	r := New()
	r.Apply(cfg.Sub("grpc.registry"))
	if len(r.opts.EtcdConfig.Endpoints) == 0 {
		t.Error("apply error")
	}
}

func TestRegistryMultiService(t *testing.T) {
	sn := "/woocoo/multi"
	cfg := testcnf.Sub("grpc")
	cfg.Parser().Set("server.namespace", sn)
	// Don't UseLogger to avoid grpclog.SetLoggerV2 caused data race
	srv := grpcx.New(grpcx.WithConfiguration(cfg))

	etcdConfg := clientv3.Config{
		Endpoints:   []string{testdata.EtcdAddr},
		DialTimeout: 10 * time.Second,
	}
	etcdCli, err := clientv3.New(etcdConfg)
	assert.NoError(t, err)
	etcdCli.Delete(context.Background(), sn, clientv3.WithPrefix())

	go func() {
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		testproto.RegisterTestServiceServer(srv.Engine(), &testproto.TestPingService{})
		if err := srv.Run(); err != nil {
			assert.NoError(t, err)
		}
	}()
	time.Sleep(time.Second * 3)
	RegisterResolver(etcdConfg)
	c, err := grpc.Dial(fmt.Sprintf("etcd://%s/", sn), grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	defer c.Close()
	hlClient := helloworld.NewGreeterClient(c)
	tsClient := testproto.NewTestServiceClient(c)
	for i := 0; i < 5; i++ {
		resp, err := hlClient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
		respts, err := tsClient.Ping(context.Background(), &testproto.PingRequest{Value: "ping"})
		assert.NoError(t, err)
		assert.Equal(t, respts.Value, "ping")
	}
	ctx, cxl := context.WithTimeout(context.Background(), time.Second*3)
	defer cxl()
	d, err := grpc.DialContext(ctx, fmt.Sprintf("etcd://%s/", sn), grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	defer d.Close()
	assert.NoError(t, err)
}

func TestRegisterResolver(t *testing.T) {
	sn := "/group/test"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{testdata.EtcdAddr},
		DialTimeout: 10 * time.Second,
	}
	reg, err := BuildFromConfig(&Options{
		EtcdConfig: etcdConfg,
		TTL:        10 * time.Minute,
	})
	assert.NoError(t, err)
	ctx, cxl := context.WithTimeout(context.Background(), time.Second)
	reg.client.Delete(ctx, sn, clientv3.WithPrefix())
	defer cxl()

	var wg sync.WaitGroup
	wg.Add(2)
	runfunc := func(namespace, name, host string, port int) {
		service := &registry.ServiceInfo{
			Namespace: namespace,
			Name:      name,
			Version:   "1.0",
			Host:      host,
			Port:      port,
			Metadata:  nil,
		}
		err := reg.Register(service)
		assert.NoError(t, err)
		l, err := net.Listen("tcp", string(service.Host)+":"+strconv.Itoa(service.Port))
		assert.NoError(t, err)
		srv := grpc.NewServer()
		helloworld.RegisterGreeterServer(srv, &helloworld.Server{})
		wg.Done()
		assert.NoError(t, srv.Serve(l))
	}

	go func() {
		runfunc(sn, "grpc1", "localhost", 9999)
	}()
	go func() {
		runfunc(sn, "grpc2", "localhost", 9998)
	}()
	wg.Wait()
	res, err := reg.client.Get(context.Background(), sn, clientv3.WithPrefix())
	assert.NoError(t, err)
	assert.EqualValues(t, res.Count, 2)
	RegisterResolver(etcdConfg)
	c, err := grpc.Dial(fmt.Sprintf("etcd://%s/", sn), grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
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
	sn := "/woocoo/registrytest"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{testdata.EtcdAddr},
		DialTimeout: 10 * time.Second,
	}
	etcdCli, err := clientv3.New(etcdConfg)
	etcdCli.Delete(context.Background(), sn, clientv3.WithPrefix())

	cfg := testcnf.Sub("grpc")
	cfg.Parser().Set("server.namespace", sn)
	cfg.Parser().Set("server.addr", ":20002")
	srv := grpcx.New(grpcx.WithConfiguration(cfg))
	go func() {
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		if err := srv.Run(); err != nil {
			assert.NoError(t, err)
		}
	}()
	cfg2 := testcnf.Sub("grpc")
	cfg2.Parser().Set("server.namespace", sn)
	cfg2.Parser().Set("server.addr", ":20003")
	srv2 := grpcx.New(grpcx.WithConfiguration(cfg2))
	go func() {
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		if err := srv2.Run(); err != nil {
			assert.NoError(t, err)
		}
	}()
	time.Sleep(time.Millisecond * 100)
	RegisterResolver(etcdConfg)
	c, err := grpc.Dial(fmt.Sprintf("etcd://%s/", sn), grpc.WithInsecure(), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	assert.NoError(t, err)
	assert.NotNil(t, c)
	defer c.Close()
	client := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {
		resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
	}
	{
		//delete one of the service,srv2 should be worked
		for _, info := range srv.ServiceInfos {
			_, err = etcdCli.Delete(context.Background(), info.BuildKey())
			assert.NoError(t, err)
		}
		time.Sleep(time.Millisecond * 100)
		_, err = client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		//todo robin validate
	}
	{
		srv.Stop()
		srv2.Stop()
		//sleep let unregistry work,the latency 500 work fine, below will failure
		time.Sleep(time.Millisecond * 500)
		resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.Nil(t, resp)
		assert.Error(t, err)
	}

}
