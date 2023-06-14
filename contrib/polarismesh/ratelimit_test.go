package polarismesh

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/testco/mock/helloworld"
	"github.com/tsingsun/woocoo/testco/wctest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func mockRatelimitRule(api *httpAPI) *httpAPI {
	url := api.baseUrl + "/naming/v1/ratelimits"
	checkRule, err := http.NewRequest(http.MethodGet, url+"?name=limit-test", nil)
	require.NoError(api.t, err)
	bd, err := api.do(checkRule)
	require.NoError(api.t, err)
	if !strings.Contains(string(bd), "id") {
		req, err := http.NewRequest(http.MethodPost, url,
			strings.NewReader(`
[{
  "name":"limit-test",
  "namespace":"ratelimit",
  "service":"helloworld.Greeter",
  "method": {"type":"EXACT","value":"SayHello"},
  "arguments":[
    {"type":"HEADER","key":"rateLimit","value":{"type":"EXACT","value":"text"}},
	{"type": "CALLER_SERVICE","key": "ratelimit","value": {"type": "EXACT","value": "helloworld.Greeter"}},
	{"type": "CALLER_IP","key": "$caller_ip","value": {"type": "EXACT","value": "127.0.0.1"}}
  ],
  "resource": "QPS",
  "type": "LOCAL",
  "disable": false,
  "amounts": [{"maxAmount": 1,"validDuration": "10s"}],
  "failover": "FAILOVER_LOCAL"
}]`))
		require.NoError(api.t, err)
		bd, err = api.do(req)
		require.NoError(api.t, err)
		require.Contains(api.t, string(bd), "execute success")
	}
	return api
}

// single machine test: rate limit 1 req/3s to sayHello. header: rateLimit=1
func TestRateLimitUnaryServerInterceptor(t *testing.T) {
	b, err := os.ReadFile("./testdata/ratelimit.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	var srv *grpcx.Server
	err = wctest.RunWait(t, time.Second*2, func() error {
		srv = grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
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
	mockRatelimitRule(meshapi(t).getToken())

	hcli := helloworld.NewGreeterClient(c)
	breaked := false
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 500)
		// Todo test pass in local server v1.72, but fail in github ci docker v1.70,so ignore it
		_, err = hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "polaris"})
		if err != nil {
			if codes.ResourceExhausted == status.Code(err) {
				breaked = true
				break
			}
		}
	}
	assert.True(t, breaked)
}
