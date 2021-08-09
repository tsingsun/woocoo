package client

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"strings"
)

type ServerConfig struct {
	Addr           string `json:"addr" yaml:"addr"`
	SSLCertificate string `json:"ssl_certificate" yaml:"ssl_certificate"`
	Location       string `json:"location" yaml:"location"`
	Version        string `json:"version" yaml:"version"`
}

type Client struct {
	serverConfig     ServerConfig
	registry         registry.Registry
	resolverBuilder  resolver.Builder
	dialOpts         []grpc.DialOption
	configuration    *conf.Configuration
	configurationKey string
}

func New(opts ...Option) *Client {
	c := &Client{}
	for _, o := range opts {
		o(c)
	}
	if c.configuration != nil && c.configurationKey != "" {
		c.Apply(c.configuration, c.configurationKey)
	}
	return c
}

func (c *Client) Apply(cfg *conf.Configuration, path string) {
	if err := cfg.Sub(path).Parser().UnmarshalByJson("server", &c.serverConfig); err != nil {
		panic(err)
	}
	if k := strings.Join([]string{path, "registry"}, conf.KeyDelimiter); cfg.IsSet(k) {
		c.registry = registry.GetRegistry(cfg.String(strings.Join([]string{path, "registry", "schema"}, conf.KeyDelimiter)))
		if ap, ok := c.registry.(conf.Configurable); ok {
			ap.Apply(cfg, k)
			c.resolverBuilder = c.registry.ResolverBuilder(c.serverConfig.Location)
			//global
			resolver.Register(c.resolverBuilder)
		}
	}
	//client
	if k := strings.Join([]string{path, "client"}, conf.KeyDelimiter); cfg.IsSet(k) {
		c.dialOpts = append(c.dialOpts, grpcDialOptions.Apply(cfg, k)...)
	}
}

func (c *Client) Dial(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	c.dialOpts = append(c.dialOpts, opts...)
	return grpc.Dial(c.resolverBuilder.Scheme()+"://"+c.serverConfig.Location+"/", c.dialOpts...)
}
