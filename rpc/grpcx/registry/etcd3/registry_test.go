package etcd3

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/test/testproto"
	"github.com/tsingsun/woocoo/testco/mock/helloworld"
	"github.com/tsingsun/woocoo/testco/wctest"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"strconv"
	"testing"
	"time"
)

var testcnf = conf.New(conf.WithLocalPath(testdata.TestConfigFile()), conf.WithBaseDir(testdata.BaseDir())).Load()

func TestRegistry_Apply(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20000
    namespace: /woocoo/service
    version: "1.0"
    ipv6: true
  registry:
    scheme: etcd
    ttl: 600s
    etcd:
      endpoints:
        - 127.0.0.1:2379
      tls:
        cert: "x509/server.crt"
        key: "x509/server.key"
      dial-timeout: 3s
      dial-keep-alive-time: 1h
`)
	cfg := conf.NewFromBytes(b)
	cfg.SetBaseDir(testdata.BaseDir())
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
	cfg.Parser().Set("server.addr", "127.0.0.1:20010")
	// Don't WithGrpcLogger to avoid grpclog.SetLoggerV2 caused data race
	srv := grpcx.New(grpcx.WithConfiguration(cfg))

	etcdConfg := clientv3.Config{
		Endpoints:   []string{testdata.EtcdAddr},
		DialTimeout: 10 * time.Second,
	}
	etcdCli, err := clientv3.New(etcdConfg)
	assert.NoError(t, err)
	ctx, cxl := context.WithTimeout(context.Background(), time.Second*1)
	defer cxl()
	_, err = etcdCli.Delete(ctx, sn, clientv3.WithPrefix())
	require.NoError(t, err)
	err = wctest.RunWait(t, time.Second*2, func() error {
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		testproto.RegisterTestServiceServer(srv.Engine(), &testproto.TestPingService{})
		return srv.Run()
	})
	require.NoError(t, err)
	RegisterResolver(etcdConfg)
	c, err := grpc.Dial(fmt.Sprintf("etcd://%s/", sn), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
	require.NoError(t, err)
	defer c.Close()
	hlClient := helloworld.NewGreeterClient(c)
	tsClient := testproto.NewTestServiceClient(c)
	for i := 0; i < 5; i++ {
		resp, err := hlClient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		require.NoError(t, err)
		assert.Equal(t, resp.Message, "Hello round robin")
		respts, err := tsClient.Ping(context.Background(), &testproto.PingRequest{Value: "ping"})
		assert.NoError(t, err)
		assert.Equal(t, respts.Value, "ping")
	}
	ctx, cxl = context.WithTimeout(context.Background(), time.Second*3)
	defer cxl()
	d, err := grpc.DialContext(ctx, fmt.Sprintf("etcd://%s/", sn), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
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
	_, _ = reg.client.Delete(ctx, sn, clientv3.WithPrefix())
	defer cxl()

	runfunc := func(namespace, name, host string, port int, listen bool) {
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
		if !listen {
			return
		}
		l, err := net.Listen("tcp", service.Host+":"+strconv.Itoa(service.Port))
		assert.NoError(t, err)
		srv := grpc.NewServer()
		helloworld.RegisterGreeterServer(srv, &helloworld.Server{})
		assert.NoError(t, srv.Serve(l))
	}
	t.Run("1-grpc-cluster", func(t *testing.T) {
		wctest.RunWait(t, time.Second*2, func() error {
			runfunc(sn, "grpc1", "127.0.0.1", 9999, true)
			return nil
		}, func() error {
			runfunc(sn, "grpc2", "127.0.0.1", 9998, true)
			return nil
		})

		res, err := reg.client.Get(context.Background(), sn, clientv3.WithPrefix())
		assert.NoError(t, err)
		assert.EqualValues(t, res.Count, 2)
		RegisterResolver(etcdConfg)
		c, err := grpc.Dial(fmt.Sprintf("etcd://%s/", sn), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
		assert.NoError(t, err)
		defer c.Close()
		client := helloworld.NewGreeterClient(c)
		for i := 0; i < 5; i++ {
			resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
			assert.NoError(t, err)
			assert.Equal(t, resp.Message, "Hello round robin")
		}
	})
	t.Run("2-exists-node", func(t *testing.T) {
		t.Logf("must run after pretest and renew reg")
		reg, err = BuildFromConfig(&Options{
			EtcdConfig: etcdConfg,
			TTL:        10 * time.Minute,
		})
		require.NoError(t, err)
		runfunc(sn, "grpc2", "127.0.0.1", 9998, false)
	})
}

func TestRegistryGrpcx(t *testing.T) {
	//gn:="group"
	sn := "/woocoo/registrytest"
	etcdConfg := clientv3.Config{
		Endpoints:   []string{testdata.EtcdAddr},
		DialTimeout: 10 * time.Second,
	}
	etcdCli, err := clientv3.New(etcdConfg)
	require.NoError(t, err)
	_, _ = etcdCli.Delete(context.Background(), sn, clientv3.WithPrefix())
	var srv, srv2 *grpcx.Server
	err = wctest.RunWait(t, time.Second, func() error {
		cfg := testcnf.Sub("grpc")
		cfg.Parser().Set("server.namespace", sn)
		cfg.Parser().Set("server.addr", "127.0.0.1:50053")
		srv = grpcx.New(grpcx.WithConfiguration(cfg))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	}, func() error {
		cfg := testcnf.Sub("grpc")
		cfg.Parser().Set("server.namespace", sn)
		cfg.Parser().Set("server.addr", "127.0.0.1:50054")
		srv2 = grpcx.New(grpcx.WithConfiguration(cfg))
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		return srv2.Run()
	})
	require.NoError(t, err)
	RegisterResolver(etcdConfg)
	c, err := grpc.Dial(fmt.Sprintf("etcd://%s/", sn),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{ "loadBalancingConfig": [{"%v": {}}] }`, roundrobin.Name)))
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
		// delete one of the service,srv2 should be worked
		for _, info := range srv.ServiceInfos {
			_, err = etcdCli.Delete(context.Background(), info.BuildKey())
			assert.NoError(t, err)
		}
		time.Sleep(time.Millisecond * 100)
		_, err = client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		// todo robin validate
	}
	{
		assert.NoError(t, srv.Stop(context.Background()))
		assert.NoError(t, srv2.Stop(context.Background()))
		// sleep let unregistry work,the latency 500 work fine, below will failure
		time.Sleep(time.Millisecond * 500)
		resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.Nil(t, resp)
		assert.Error(t, err)
	}
}
