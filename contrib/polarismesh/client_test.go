package polarismesh

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/internal/mock/helloworld"
	"github.com/tsingsun/woocoo/internal/wctest"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func init() {
	log := grpclog.NewLoggerV2WithVerbosity(io.Discard, os.Stdout, os.Stdout, 5)
	grpclog.SetLoggerV2(log)
}

type httpAPI struct {
	user, pwd, token string
	baseUrl          string
	t                *testing.T
}

func meshapi(t *testing.T) *httpAPI {
	return &httpAPI{
		user:    "polaris",
		pwd:     "polaris",
		baseUrl: "http://localhost:8090",
		t:       t,
	}
}

func (api *httpAPI) getToken() *httpAPI {
	resp, err := http.Post(fmt.Sprintf("%s/core/v1/user/login", api.baseUrl),
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"name":"%s","password":"%s"}`, api.user, api.pwd)))
	require.NoError(api.t, err)
	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	require.NoError(api.t, err)
	var token struct {
		LoginResponse struct {
			Token string `json:"token"`
		} `json:"loginResponse"`
	}
	require.NoError(api.t, json.Unmarshal(bs, &token))
	api.token = token.LoginResponse.Token
	return api
}

func (api *httpAPI) routings() *httpAPI {
	checkreq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/naming/v2/routings?id=%s", api.baseUrl, "cn"), nil)
	data, err := api.do(checkreq)
	require.NoError(api.t, err)
	if strings.Contains(string(data), "routing-test") {
		return api
	}
	// one route ,one guarantee
	payload := strings.NewReader(`
[{
  "id": "cn",
  "name": "routing-test",
  "enable": true,
  "description": "",
  "priority": 0,
  "routing_config": {
    "@type": "type.googleapis.com/v2.RuleRoutingConfig",
    "sources": [{"service": "*","namespace": "routingTest"}],
    "destinations": [{"service": "helloworld.Greeter","namespace": "routingTest"}],
    "rules": [{
      "name": "规则0",
      "sources": [{
        "service": "*",
        "namespace": "routingTest",
        "arguments": [{
          "type": "HEADER",
          "key": "country",
	      "value": {"type": "EXACT","value": "CN","value_type": "TEXT"}
        }]
      }],
      "destinations": [{
        "service": "helloworld.Greeter",
        "namespace": "routingTest",
        "labels": {
            "location": {"value": "amoy","type": "EXACT","value_type": "TEXT"}
        },
        "weight": 100,
        "isolate": false,
        "name": "group-0"
      }]
    }]
  }
},{
  "id": "us",
  "name": "routing-test",
  "enable": true,
  "description": "",
  "priority": 0,
  "routing_config": {
    "@type": "type.googleapis.com/v2.RuleRoutingConfig",
    "sources": [{"service": "*","namespace": "routingTest"}],
    "destinations": [{"service": "helloworld.Greeter","namespace": "routingTest"}],
    "rules": [{
      "name": "规则0",
      "sources": [{
        "service": "*",
        "namespace": "routingTest",
        "arguments": [{
          "type": "HEADER",
          "key": "country",
	      "value": {"type": "NOT_EQUALS","value": "CN","value_type": "TEXT"}
        }]
      }],
      "destinations": [{
        "service": "helloworld.Greeter",
        "namespace": "routingTest",
        "labels": {
            "location": {"value": "us","type": "EXACT","value_type": "TEXT"}
        },
        "weight": 100,
        "isolate": false,
        "name": "group-0"
      }]
    }]
  }
},{
  "id": "guarantee",
  "name": "guarantee",
  "enable": true,
  "description": "",
  "priority": 1,
  "routing_config": {
    "@type": "type.googleapis.com/v2.RuleRoutingConfig",
    "sources": [{"service": "*","namespace": "routingTest"}],
    "destinations": [{"service": "helloworld.Greeter","namespace": "routingTest"}],
    "rules": [{
      "name": "规则0",
      "sources": [{
        "service": "*",
        "namespace": "*",
        "arguments": []
      }],
      "destinations": [{
        "service": "helloworld.Greeter",
        "namespace": "routingTest",
        "labels": {
            "location": {"value": "amoy","type": "NOT_EQUALS","value_type": "TEXT"}
        },
        "weight": 100,
        "isolate": false,
        "name": "group-0"
      }]
    }]
  }
}]
`)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/naming/v2/routings", api.baseUrl), payload)
	require.NoError(api.t, err)
	_, err = api.do(req)
	require.NoError(api.t, err)
	return api
}

func (api *httpAPI) routingsEnable(id string, bl bool) *httpAPI {
	payload := strings.NewReader(fmt.Sprintf(`[{"id":"%s","enable":%s}]`, id, strconv.FormatBool(bl)))
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/naming/v2/routings/enable", api.baseUrl), payload)
	require.NoError(api.t, err)
	bd, err := api.do(req)
	require.NoError(api.t, err)
	require.Contains(api.t, string(bd), "execute success")
	return api
}

func (api *httpAPI) rateLimit() *httpAPI {
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
  "arguments":[{
     "type":"HEADER","key":"rateLimit","value":{"type":"EXACT","value":"text"}
  }],
  "resource": "QPS",
  "type": "LOCAL",
  "disable": false,
  "amounts": [{"maxAmount": 1,"validDuration": "3s"}],
  "failover": "FAILOVER_LOCAL"
}]`))
		require.NoError(api.t, err)
		bd, err = api.do(req)
		require.NoError(api.t, err)
		require.Contains(api.t, string(bd), "execute success")
	}
	return api
}

func (api *httpAPI) circuitBreaker() *httpAPI {
	url := api.baseUrl + "/naming/v1/circuitbreaker/rules"
	checkRule, err := http.NewRequest(http.MethodGet, url+"?name=circuitBreaker-test", nil)
	require.NoError(api.t, err)
	bd, err := api.do(checkRule)
	require.NoError(api.t, err)
	if strings.Contains(string(bd), "id") {
		return api
	}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(`
[
    {	
        "name": "circuitBreaker-test",
        "enable": true,
        "level": "METHOD",
        "description": "",
        "ruleMatcher": {
            "source": {
                "service": "*",
                "namespace": "*"
            },
            "destination": {
                "service": "helloworld.Greeter",
                "namespace": "circuitBreakerTest",
                "method": {
                    "type": "EXACT",
                    "value": "SayHello"
                }
            }
        },
        "errorConditions": [
            {
                "inputType": "RET_CODE",
                "condition": {
                    "type": "NOT_EQUALS",
                    "value": "0"
                }
            }
        ],
        "triggerCondition": [
            {
                "triggerType": "ERROR_RATE",
                "errorCount": 10,
                "errorPercent": 20,
                "interval": 5,
                "minimumRequest": 5
            }
        ],
        "recoverCondition": {
            "sleepWindow": 5,
            "consecutiveSuccess": 3
        },
        "faultDetectConfig": {
            "enable": false
        },
        "fallbackConfig": {
            "enable": true,
            "response": {
                "code": 8,
                "headers": [
                    {
                        "key": "X-CircuitBreaker-Retry",
                        "value": "5s"
                    }
                ],
                "body": "CircuitBreaker"
            }
        }
    }
]
`))
	require.NoError(api.t, err)
	bd, err = api.do(req)
	require.NoError(api.t, err)
	require.Contains(api.t, string(bd), "execute success")
	return api
}

func (api *httpAPI) do(r *http.Request) (data []byte, err error) {
	client := http.DefaultClient
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		r.Header.Add("Content-Type", "application/json")
	}
	r.Header.Add("X-Polaris-Token", api.token)
	resp, err := client.Do(r)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http status code %d", resp.StatusCode)
		return
	}
	data, err = io.ReadAll(resp.Body)
	if r.Method != http.MethodGet {
		// await for effect
		time.Sleep(time.Second)
	}
	return
}

func TestClient_Dial(t *testing.T) {
	b, err := os.ReadFile("./testdata/dialtest.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	err = wctest.RunWait(t, time.Second*2, func() error {
		srv := grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	})
	require.NoError(t, err)
	cli := grpcx.NewClient(cfg.Sub("grpc"))
	conn, err := cli.Dial("")
	require.NoError(t, err)
	require.NotNil(t, conn)
	gclient := helloworld.NewGreeterClient(conn)
	resp, err := gclient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "world"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_DialMultiServerAndDown(t *testing.T) {
	b, err := os.ReadFile("./testdata/multidown.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	var srv, srv2 *grpcx.Server
	err = wctest.RunWait(t, time.Second*2, func() error {
		opts := []grpc.ServerOption{
			grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return &helloworld.HelloReply{
					Message: "server1",
				}, nil
			}),
		}
		srv = grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	}, func() error {
		cfg2 := cfg.Sub("grpc")
		cfg2.Parser().Set("server.addr", "127.0.0.1:21113")
		opts := []grpc.ServerOption{
			grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return &helloworld.HelloReply{
					Message: "server2",
				}, nil
			}),
		}
		srv2 = grpcx.New(grpcx.WithConfiguration(cfg2), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		return srv2.Run()
	})
	require.NoError(t, err)
	cli := grpcx.NewClient(cfg.Sub("grpc"))
	c, err := cli.Dial("")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, c)
	defer c.Close()
	hcli := helloworld.NewGreeterClient(c)
	t.Run("loadbalance", func(t *testing.T) {
		var s1count, s2count int
		for i := 0; i < 10; i++ {
			resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
			assert.NoError(t, err)
			if resp.Message != "server1" {
				s1count++
			}
			if resp.Message != "server2" {
				s2count++
			}
		}
		// router robin
		assert.NotZero(t, s1count, "server1 request")
		assert.NotZero(t, s2count, "server2 request")
	})
	t.Run("down 1/2", func(t *testing.T) {
		srv.Stop(context.Background())
		time.Sleep(time.Second * 1)
		resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.NoError(t, err)
		assert.Equal(t, resp.Message, "server2")

	})
	t.Run("down 2/2", func(t *testing.T) {
		srv2.Stop(context.Background())
		//sleep let unregistry work,the latency 500 work fine, below will failure
		resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "round robin"})
		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func TestClientRouting(t *testing.T) {
	b, err := os.ReadFile("./testdata/routing.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	var (
		srv, srv2, srv3 *grpcx.Server
		expectedMsg     = "match success"
	)
	err = wctest.RunWait(t, time.Second*2, func() error {
		srv = grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	}, func() error {
		cfg2 := cfg.Sub("grpc2")
		opts := []grpc.ServerOption{
			grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return &helloworld.HelloReply{
					Message: expectedMsg,
				}, nil
			}),
		}
		srv2 = grpcx.New(grpcx.WithConfiguration(cfg2), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		return srv2.Run()
	}, func() error {
		cfg3 := cfg.Sub("grpc3")
		opts := []grpc.ServerOption{
			grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return nil, status.Error(codes.FailedPrecondition, "")
			}),
		}
		srv3 = grpcx.New(grpcx.WithConfiguration(cfg3), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv3.Engine(), &helloworld.Server{})
		return srv3.Run()
	})
	require.NoError(t, err)

	api := meshapi(t)
	api.getToken().routings()

	t.Run("native-dial-without-src-service", func(t *testing.T) {
		// api.routingsEnable("guarantee", false)
		conn, err := grpc.Dial(scheme+"://routingTest/helloworld.Greeter?route=true",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithResolvers(&resolverBuilder{}),
			grpc.WithDefaultServiceConfig(`{ "loadBalancingConfig": [{"polaris": {}}] }`),
		)
		require.NoError(t, err)
		gcli := helloworld.NewGreeterClient(conn)
		_, err = gcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "hello"})
		assert.NoError(t, err)

		// route
		for i := 0; i < 5; i++ {
			ctx := metadata.AppendToOutgoingContext(context.Background(), "country", "CN")
			resp, err := gcli.SayHello(ctx, &helloworld.HelloRequest{Name: "hello"})
			require.NoError(t, err)
			assert.Equal(t, expectedMsg, resp.Message)
		}
	})

	t.Run("route-rule-match", func(t *testing.T) {
		cli := grpcx.NewClient(cfg.Sub("grpc"))
		c, err := cli.Dial("")
		require.NoError(t, err)
		assert.NotNil(t, c)
		defer c.Close()
		hcli := helloworld.NewGreeterClient(c)
		for i := 0; i < 5; i++ {
			time.Sleep(time.Millisecond * 100)
			resp, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "match"})
			if assert.NoError(t, err) {
				// Todo test pass in local server v1.72, but fail in github ci docker v1.70,so ignore it
				assert.Equal(t, expectedMsg, resp.Message)
			}
		}
	})
}

func TestClientCircleBreaker(t *testing.T) {
	b, err := os.ReadFile("./testdata/circuitbreaker.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	var (
		makeErr = true
	)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			if makeErr {
				return nil, status.Error(codes.Canceled, "canceled")
			}
			return handler(ctx, req)
		}),
	}

	var srv, srv2 *grpcx.Server
	err = wctest.RunWait(t, time.Second*2, func() error {
		srv = grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	}, func() error {
		cfg2 := cfg.Sub("grpc2")
		srv2 = grpcx.New(grpcx.WithConfiguration(cfg2), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		return srv2.Run()
	})
	require.NoError(t, err)

	meshapi(t).getToken().circuitBreaker()

	cli := grpcx.NewClient(cfg.Sub("grpc"))
	c, err := cli.Dial("")
	require.NoError(t, err)
	assert.NotNil(t, c)
	defer c.Close()
	hcli := helloworld.NewGreeterClient(c)
	for i := 0; i < 12; i++ {
		_, err := hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "match"})
		if i == 9 {
			makeErr = false
		}
		if !makeErr {
			// TODO need implement
			// assert.Equal(t, status.Code(err).String(), codes.ResourceExhausted.String())
		} else {
			assert.Error(t, err, i)
		}
		time.Sleep(time.Millisecond * 100)

	}
	// wait for prometheus
	time.Sleep(time.Second * 2)
}
