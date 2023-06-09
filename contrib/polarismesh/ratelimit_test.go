package polarismesh

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
	"testing"
	"time"
)

// single machine test: rate limit 1 req/s to sayHello. header: rateLimit=1
func TestRateLimitUnaryServerInterceptor(t *testing.T) {
	b, err := os.ReadFile("./testdata/ratelimit.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	var srv *grpcx.Server
	err = wctest.RunWait(t, time.Second*2, func() error {
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
	// limit rule setup
	meshapi(t).getToken().rateLimit()

	hcli := helloworld.NewGreeterClient(c)
	for i := 0; i < 5; i++ {
		//time.Sleep(time.Millisecond * 200)
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("rateLimit", "text"))
		_, err := hcli.SayHello(ctx, &helloworld.HelloRequest{Name: "polaris"})
		if i > 2 {
			assert.Equal(t, codes.ResourceExhausted.String(), status.Code(err).String())
		}
	}
}
