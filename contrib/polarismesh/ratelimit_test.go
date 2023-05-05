package polarismesh

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

// single machine test: rate limit 1 req/s to sayHello. header: rateLimit=1
func TestRateLimitUnaryServerInterceptor(t *testing.T) {
	b := []byte(`
namespace: woocoo
grpc:
  server:
    addr: 127.0.0.1:20012
  engine:
    - unaryInterceptors:
        - recovery:      
        - polarisRateLimit:
  registry:
    scheme: polaris
    ttl: 600s
    polaris: 
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
        statReporter:
          enable: true
          chain:
            - prometheus
          plugin:
            prometheus:
              metricPort: 0
  client:
    target:
      namespace: woocoo
      serviceName: helloworld.Greeter
      metadata:  
        src_rateLimit: 1
    dialOption:
      - tls:
      - block:
      - timeout: 1s
      - serviceConfig: '{ "loadBalancingConfig": [{"polaris": {}}] }' 
`)
	cfg := conf.NewFromBytes(b)
	var srv *grpcx.Server
	err := wctest.RunWait(t, time.Second*5, func() error {
		srv = grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")), grpcx.WithGrpcLogger())
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	})
	require.NoError(t, err)
	cli := grpcx.NewClient(cfg.Sub("grpc"))
	c, err := cli.Dial("")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, c)
	defer func() {
		c.Close()
		srv.Stop(context.Background())
	}()
	hcli := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond * 200)
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("rateLimit", "text"))
		resp, err := hcli.SayHello(ctx, &helloworld.HelloRequest{Name: "polaris"})
		if i == 0 {
			assert.NoError(t, err)
			assert.NotNil(t, resp)
		} else {
			assert.Error(t, err)
		}
	}
}
