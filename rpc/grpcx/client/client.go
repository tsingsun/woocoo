package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"strings"
	"time"
)

type ServerConfig struct {
	Addr      string `json:"addr" yaml:"addr"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Version   string `json:"version" yaml:"version"`
}

type Client struct {
	dialOpts      registry.DialOptions
	serverConfig  ServerConfig
	configuration *conf.Configuration

	// for dialcontext
	timeout time.Duration
	// registry scheme
	scheme string
}

func New(opts ...Option) *Client {
	c := &Client{}
	for _, o := range opts {
		o(c)
	}
	c.Apply(c.configuration)
	return c
}

func (c *Client) Apply(cfg *conf.Configuration) {
	if err := cfg.Parser().Unmarshal("server", &c.serverConfig); err != nil {
		panic(err)
	}
	if k := strings.Join([]string{"registry"}, conf.KeyDelimiter); cfg.IsSet(k) {
		c.scheme = cfg.String(strings.Join([]string{"registry", "scheme"}, conf.KeyDelimiter))
		drv, ok := registry.GetRegistry(c.scheme)
		if !ok {
			panic(fmt.Errorf("registry driver not found:%s", c.scheme))
		}
		rb, err := drv.ResolverBuilder(cfg.Sub(k))
		if err != nil {
			panic(err)
		}
		c.dialOpts.GRPCDialOptions = append(c.dialOpts.GRPCDialOptions, grpc.WithResolvers(rb))
	}
	// target section is registry info for client
	if k := strings.Join([]string{"client", "target"}, conf.KeyDelimiter); cfg.IsSet(k) {
		if err := cfg.Sub(k).Unmarshal(&c.dialOpts); err != nil {
			panic(err)
		}
	}
	// grpc dial options
	if k := strings.Join([]string{"client", "grpcDialOption"}, conf.KeyDelimiter); cfg.IsSet(k) {
		c.dialOpts.GRPCDialOptions = append(c.dialOpts.GRPCDialOptions, grpcDialOptions.Apply(c, cfg, k)...)
	}
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
		target = c.targetPrefix() + c.dialOpts.ServiceName
	} else if !strings.HasPrefix(target, c.targetPrefix()) {
		return grpc.DialContext(ctx, target, append(c.dialOpts.GRPCDialOptions, opts...)...)
	}
	// attach service info
	jsonstr, err := json.Marshal(c.dialOpts)
	if err != nil {
		return nil, fmt.Errorf("DialContext:marshal dial options error:%v", err)
	}
	endpoint := base64.URLEncoding.EncodeToString(jsonstr)
	target = fmt.Sprintf("%s?%s=%s", target, registry.OptionKey, endpoint)
	return grpc.DialContext(ctx, target, append(c.dialOpts.GRPCDialOptions, opts...)...)
}
