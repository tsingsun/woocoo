package polarismesh

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/mock/helloworld"
	"github.com/tsingsun/woocoo/test/wctest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func init() {
	logger := grpclog.NewLoggerV2WithVerbosity(io.Discard, os.Stdout, os.Stdout, 5)
	grpclog.SetLoggerV2(logger)
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
	checkreq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/naming/v2/routings?id=%s", api.baseUrl, "byCountry"), nil)
	data, err := api.do(checkreq)
	require.NoError(api.t, err)
	if strings.Contains(string(data), "routing-test") {
		return api
	}
	// one route ,one guarantee
	payload, err := os.Open("./testdata/routing.json")
	require.NoError(api.t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/naming/v2/routings", api.baseUrl), payload)
	require.NoError(api.t, err)
	bd, err := api.do(req)
	require.NoError(api.t, err)
	require.Contains(api.t, string(bd), "execute success")
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
                "namespace": "circuitBreakerTest"
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
                "errorPercent": 1,
                "interval": 30,
                "minimumRequest": 5
            }
        ],
        "recoverCondition": {
            "sleepWindow": 30,
            "consecutiveSuccess": 3
        },
        "faultDetectConfig": {
            "enable": false
        },
        "fallbackConfig": {
            "enable": true,
            "response": {
                "code": 429,
                "headers": [
                    {
                        "key": "X-CircuitBreaker-Retry",
                        "value": "30s"
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
	err = wctest.RunWait(t.Log, time.Second*2, func() error {
		srv := grpcx.New(grpcx.WithConfiguration(cfg.Sub("grpc")))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	})
	require.NoError(t, err)
	t.Run("normal", func(t *testing.T) {
		cli, err := grpcx.NewClient(cfg.Sub("grpc"))
		require.NoError(t, err)
		conn, err := cli.Dial("")
		require.NoError(t, err)
		require.NotNil(t, conn)
		gclient := helloworld.NewGreeterClient(conn)
		resp, err := gclient.SayHello(context.Background(), &helloworld.HelloRequest{Name: "world"})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("miss service", func(t *testing.T) {
		sc := cfg.Sub("grpc")
		sc.Parser().Set("client.target.serviceName", "")
		cli, err := grpcx.NewClient(sc)
		require.NoError(t, err)
		_, err = cli.Dial("")
		assert.ErrorContains(t, err, "resolver need a target host or service name")
	})
	t.Run("miss skdctx", func(t *testing.T) {
		sc := cfg.Sub("grpc")
		target := fmt.Sprintf("%s://%s/%s", scheme, sc.String("client.target.namespace"), sc.String("client.target.serviceName"))
		rb := &resolverBuilder{}
		_, err = grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithResolvers(rb))
		assert.NoError(t, err)
	})
	t.Run("wrong target", func(t *testing.T) {
		sc := cfg.Sub("grpc")
		cli, err := grpcx.NewClient(sc)
		require.NoError(t, err)
		target := fmt.Sprintf("%s://%s/%s", scheme, "127.0.0.1:abc", "errorPort")
		_, err = cli.Dial(target)
		assert.Error(t, err)
	})
}

func TestClient_DialMultiServerAndDown(t *testing.T) {
	b, err := os.ReadFile("./testdata/multidown.yaml")
	require.NoError(t, err)
	cfg := conf.NewFromBytes(b)
	var srv, srv2 *grpcx.Server
	err = wctest.RunWait(t.Log, time.Second*2, func() error {
		opts := []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
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
			grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
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
	cli, err := grpcx.NewClient(cfg.Sub("grpc"))
	require.NoError(t, err)
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
	cnf := conf.NewFromBytes(b)
	var (
		srvdf, srv2amoy, srv3us *grpcx.Server
		expectedMsg             = "match success"
	)
	err = wctest.RunWait(t.Log, time.Second*2, func() error {
		// guarantee default
		srvdf = grpcx.New(grpcx.WithConfiguration(cnf.Sub("grpc")))
		helloworld.RegisterGreeterServer(srvdf.Engine(), &helloworld.Server{})
		return srvdf.Run()
	}, func() error {
		// cn match amoy
		cfg := cnf.Sub("grpc2")
		opts := []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return &helloworld.HelloReply{
					Message: expectedMsg,
				}, nil
			}),
		}
		srv2amoy = grpcx.New(grpcx.WithConfiguration(cfg), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv2amoy.Engine(), &helloworld.Server{})
		return srv2amoy.Run()
	}, func() error {
		// not cn match us
		cfg := cnf.Sub("grpc3")
		opts := []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return nil, status.Error(codes.FailedPrecondition, "")
			}),
		}
		srv3us = grpcx.New(grpcx.WithConfiguration(cfg), grpcx.WithGrpcOption(opts...))
		helloworld.RegisterGreeterServer(srv3us.Engine(), &helloworld.Server{})
		return srv3us.Run()
	})
	require.NoError(t, err)

	api := meshapi(t)
	api.getToken().routings()

	drv, ok := registry.GetRegistry(scheme)
	require.True(t, ok)
	rgcnf := cnf.Sub("routingRegistry")
	rgcnf.Parser().Set("ref", "routingRegistry")
	rb, err := drv.ResolverBuilder(rgcnf)
	balancer.Register(NewBalancerBuilder("routing", getPolarisContext(t, "routingRegistry")))

	t.Run("native-dial-with-src-service", func(t *testing.T) {
		require.NoError(t, err)
		// api.routingsEnable("guarantee", false)
		conn, err := grpc.Dial(scheme+"://routingTest/helloworld.Greeter?route=true&srcservice=helloworld.Greeter",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithResolvers(rb),
			grpc.WithDefaultServiceConfig(`{ "loadBalancingConfig": [{"routing": {}}] }`),
		)
		require.NoError(t, err)
		require.NotNil(t, conn)
		defer conn.Close()
		gcli := helloworld.NewGreeterClient(conn)
		// don't have header will not match any route
		ctx := metadata.AppendToOutgoingContext(context.Background(), "country", "Not")
		_, err = gcli.SayHello(ctx, &helloworld.HelloRequest{Name: "hello"})
		require.NoError(t, err)
		// route
		for i := 0; i < 5; i++ {
			ctx := metadata.AppendToOutgoingContext(context.Background(), "country", "CN")
			resp, err := gcli.SayHello(ctx, &helloworld.HelloRequest{Name: "hello"})
			require.NoError(t, err)
			assert.Equal(t, expectedMsg, resp.Message)
		}
	})
	t.Run("native-dial-with-src-service-meta", func(t *testing.T) {
		target := fmt.Sprintf("%s://routingTest/helloworld.Greeter?route=true&srcservice=helloworld.Greeter&%s=%s",
			scheme, "src_custom", "custom")
		conn, err := grpc.Dial(target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithResolvers(rb),
			grpc.WithDefaultServiceConfig(`{ "loadBalancingConfig": [{"routing": {}}] }`),
		)
		require.NoError(t, err)
		require.NotNil(t, conn)
		defer conn.Close()
		gcli := helloworld.NewGreeterClient(conn)
		resp, err := gcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "customer"})
		require.NoError(t, err)
		assert.Contains(t, resp.Message, "customer")
	})
	t.Run("route-rule-match-cn", func(t *testing.T) {
		cli, err := grpcx.NewClient(cnf.Sub("grpc"))
		require.NoError(t, err)
		conn, err := cli.Dial("", grpc.WithDefaultServiceConfig(`{ "loadBalancingConfig": [{"routing": {}}] }`))
		require.NoError(t, err)
		require.NotNil(t, conn)
		defer conn.Close()
		hcli := helloworld.NewGreeterClient(conn)
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

// TestClientCircleBreaker test circuit breaker, it use independent polaris config
func TestClientCircleBreaker(t *testing.T) {
	b, err := os.ReadFile("./testdata/circuitbreaker.yaml")
	require.NoError(t, err)
	cnf := conf.NewFromBytes(b)
	var srv, srv2 *grpcx.Server
	err = wctest.RunWait(t.Log, time.Second*2, func() error {
		count := 0
		srv = grpcx.New(grpcx.WithConfiguration(cnf.Sub("grpc")), grpcx.WithGrpcOption(
			grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				count++
				//log.Print(count)
				return nil, nil
			})))
		helloworld.RegisterGreeterServer(srv.Engine(), &helloworld.Server{})
		return srv.Run()
	}, func() error {
		cfg := cnf.Sub("grpc2")
		srv2 = grpcx.New(grpcx.WithConfiguration(cfg), grpcx.WithGrpcOption(
			grpc.ChainUnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				return nil, status.Error(codes.Canceled, "canceled")
			})))
		helloworld.RegisterGreeterServer(srv2.Engine(), &helloworld.Server{})
		return srv2.Run()
	})
	require.NoError(t, err)

	meshapi(t).getToken().circuitBreaker()

	cli, err := grpcx.NewClient(cnf.Sub("grpc"))
	require.NoError(t, err)
	balancer.Register(NewBalancerBuilder("circuit_breaker", getPolarisContext(t, "circuitBreakerRegistry")))
	c, err := cli.Dial("", grpc.WithDefaultServiceConfig(`{ "loadBalancingConfig": [{"circuit_breaker": {}}] }`))
	require.NoError(t, err)
	assert.NotNil(t, c)
	defer c.Close()
	hcli := helloworld.NewGreeterClient(c)

	// make cb
	errcount := 0
	for i := 0; i < 6; i++ {
		for i := 0; i < 10; i++ {
			_, err = hcli.SayHello(context.Background(), &helloworld.HelloRequest{Name: "match"})
			if err != nil {
				errcount++
			}
		}
		time.Sleep(time.Second * 1)
		log.Println("batch:", i, "errcount:", errcount)
	}
	log.Println("make cb done")
	// TODO error count is not stable, it is less than robbin algorithm and circuit break will work in later request,
	// but in github ci may not work.
	assert.Less(t, errcount, 30)
}
