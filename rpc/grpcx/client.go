package grpcx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
	"time"
)

type ServerConfig struct {
	Addr      string `json:"addr" yaml:"addr"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Version   string `json:"version" yaml:"version"`
}

// Client is a grpc client helper,build a grpc client connection by configuration with registry.
// the configuration is like this:
//
//	server:  for grpc server info
//	registry:  for registry center info
//	client:  for grpc client info
//	  target:  for grpc service info using with registry or dial directly
type Client struct {
	registryOptions registry.DialOptions
	serverConfig    ServerConfig
	dialOptions     []grpc.DialOption
	// for dialcontext
	timeout time.Duration
	// registry scheme
	scheme string
	// if withTransportCredentials is false, auto client auto init with insecure
	withTransportCredentials bool
}

func NewClient(cfg *conf.Configuration) *Client {
	c := &Client{}
	c.registryOptions.Namespace = cfg.Root().Namespace()
	c.Apply(cfg)
	return c
}

func (c *Client) Apply(cfg *conf.Configuration) {
	// server info
	if err := cfg.Parser().Unmarshal("server", &c.serverConfig); err != nil {
		panic(err)
	}
	// target info
	if k := conf.Join("client", "target"); cfg.IsSet(k) {
		if err := cfg.Sub(k).Unmarshal(&c.registryOptions); err != nil {
			panic(err)
		}
	}

	var pluginOptions []grpc.DialOption
	// config dial options, lowest priority
	if k := conf.Join("client", "dialOption"); cfg.IsSet(k) {
		pluginOptions = append(pluginOptions, optionsManager.BuildDialOption(c, cfg, k)...)
	}
	if !c.withTransportCredentials {
		// make sure put first, thus user can overwrite it
		pluginOptions = append(pluginOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
		c.withTransportCredentials = true
	}
	// if using registry
	if k := "registry"; cfg.IsSet(k) {
		c.scheme = cfg.String(conf.Join(k, "scheme"))
		drv, ok := registry.GetRegistry(c.scheme)
		if !ok {
			panic(fmt.Errorf("registry driver not found:%s", c.scheme))
		}
		rb, err := drv.ResolverBuilder(cfg.Sub(k))
		if err != nil {
			panic(err)
		}
		rdo, err := drv.WithDialOptions(c.registryOptions)
		if err != nil {
			panic(err)
		}
		pluginOptions = append(pluginOptions, rdo...)
		pluginOptions = append(pluginOptions, grpc.WithResolvers(rb))
	}
	// custom dial options, highest priority
	c.dialOptions = append(pluginOptions, c.dialOptions...)
}

func (c *Client) targetPrefix() string {
	if c.scheme == "" {
		return ""
	}
	return c.scheme + "://"
}

// Dial creates a gRPC client connection with the given target,and covert to DialContext if client.timeout > 0
func (c *Client) Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if c.timeout == 0 {
		return c.DialContext(context.Background(), target, opts...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.DialContext(ctx, target, opts...)
}

// DialContext creates a gRPC client connection with the given target.
//
// The target will be parsed as a URL.your resolver must parse the target.
func (c *Client) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if target == "" {
		target = c.targetPrefix() + c.registryOptions.ServiceName
	} else if !strings.HasPrefix(target, c.targetPrefix()) {
		return grpc.DialContext(ctx, target, append(c.dialOptions, opts...)...)
	}
	// attach service info
	jsonstr, err := json.Marshal(c.registryOptions)
	if err != nil {
		return nil, fmt.Errorf("DialContext:marshal dial options error:%v", err)
	}
	endpoint := base64.URLEncoding.EncodeToString(jsonstr)
	target = fmt.Sprintf("%s?%s=%s", target, registry.OptionKey, endpoint)
	return grpc.DialContext(ctx, target, append(c.dialOptions, opts...)...)
}
