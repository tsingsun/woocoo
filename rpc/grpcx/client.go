package grpcx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func NewClient(cfg *conf.Configuration) (*Client, error) {
	c := &Client{}
	c.registryOptions.Namespace = cfg.Root().Namespace()
	if err := c.Apply(cfg); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) Apply(cfg *conf.Configuration) error {
	// server info
	if err := cfg.Parser().Unmarshal("server", &c.serverConfig); err != nil {
		return err
	}
	// target info
	if k := conf.Join("client", "target"); cfg.IsSet(k) {
		if err := cfg.Sub(k).Unmarshal(&c.registryOptions); err != nil {
			return err
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
			return fmt.Errorf("registry driver not found:%s", c.scheme)
		}
		rb, err := drv.ResolverBuilder(cfg.Sub(k))
		if err != nil {
			return err
		}
		rdo, err := drv.WithDialOptions(c.registryOptions)
		if err != nil {
			return err
		}
		// let registry defined can be overridden by user customer, but make registry resolver as default.
		pluginOptions = append(rdo, pluginOptions...)
		pluginOptions = append(pluginOptions, grpc.WithResolvers(rb))
	}
	// custom dial options, the highest priority
	c.dialOptions = append(pluginOptions, c.dialOptions...)
	return nil
}

func (c *Client) targetPrefix() string {
	if c.scheme == "" {
		return ""
	}
	return c.scheme + "://"
}

// Dial creates a gRPC client connection with the given target, and covert to DialContext if client.timeout > 0
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
	tarp := c.targetPrefix()
	if target == "" {
		if tarp != "" {
			// use registry
			target = tarp + c.registryOptions.ServiceName
		} else {
			target = c.serverConfig.Addr
			return grpc.DialContext(ctx, target, append(c.dialOptions, opts...)...)
		}
	} else if tarp == "" || !strings.HasPrefix(target, tarp) {
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
