package client

import (
	"context"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"strings"
	"time"
)

type ServerConfig struct {
	Addr           string `json:"addr" yaml:"addr"`
	SSLCertificate string `json:"ssl_certificate" yaml:"ssl_certificate"`
	Location       string `json:"location" yaml:"location"`
	Version        string `json:"version" yaml:"version"`
}

type Client struct {
	serverConfig    ServerConfig
	registry        registry.Registry
	resolverBuilder resolver.Builder
	dialOpts        []grpc.DialOption
	configuration   *conf.Configuration

	// for dialcontext
	timeout time.Duration
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
		c.registry = registry.GetRegistry(cfg.String(strings.Join([]string{"registry", "schema"}, conf.KeyDelimiter)))
		if ap, ok := c.registry.(conf.Configurable); ok {
			ap.Apply(cfg.Sub(k))
			c.resolverBuilder = c.registry.ResolverBuilder(c.serverConfig.Location)
			//global
			resolver.Register(c.resolverBuilder)
		}
	}
	//client
	if k := strings.Join([]string{"client"}, conf.KeyDelimiter); cfg.IsSet(k) {
		c.dialOpts = append(c.dialOpts, grpcDialOptions.Apply(c, cfg, k)...)
	}
}

func (c *Client) Dial(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	c.dialOpts = append(c.dialOpts, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return grpc.DialContext(ctx, c.resolverBuilder.Scheme()+"://"+c.serverConfig.Location+"/", c.dialOpts...)
}
