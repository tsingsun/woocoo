package client

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
)

var (
	grpcDialOptions = newConfigurableGrpcDialOptions()
)

func init() {
	registerInternal()
}

type Option func(client *Client)

func Configuration(configuration *conf.Configuration, configurationKey string) Option {
	return func(c *Client) {
		c.configuration = configuration
		c.configurationKey = configurationKey
	}
}

type configurableGrpcClientOptions struct {
	do   map[string]func(*conf.Configuration) grpc.DialOption
	ucit map[string]func(*conf.Configuration) grpc.UnaryClientInterceptor
	scit map[string]func(*conf.Configuration) grpc.StreamClientInterceptor
}

func newConfigurableGrpcDialOptions() *configurableGrpcClientOptions {
	return &configurableGrpcClientOptions{
		do:   make(map[string]func(*conf.Configuration) grpc.DialOption),
		ucit: make(map[string]func(*conf.Configuration) grpc.UnaryClientInterceptor),
		scit: make(map[string]func(*conf.Configuration) grpc.StreamClientInterceptor),
	}
}

func (c configurableGrpcClientOptions) unaryInterceptorHandler(cnf *conf.Configuration) grpc.DialOption {
	var opts []grpc.UnaryClientInterceptor
	its := cnf.SubOperator("")
	for _, it := range its {
		var name string
		for s, _ := range it.Raw() {
			name = s
			break
		}
		if handler, ok := c.ucit[name]; ok {
			itcfg := cnf.CutFromOperator(it)
			opts = append(opts, handler(itcfg))
		}
	}
	return grpc.WithChainUnaryInterceptor(opts...)
}

func (c configurableGrpcClientOptions) Apply(cfg *conf.Configuration, path string) (opts []grpc.DialOption) {
	hfs := cfg.ParserOperator().Slices(path)
	for _, hf := range hfs {
		var name string
		for s, _ := range hf.Raw() {
			name = s
			break
		}
		if name == "unaryInterceptors" {
			itcfg := cfg.CutFromOperator(hf)
			opts = append(opts, c.unaryInterceptorHandler(itcfg))
		}
		if handler, ok := c.do[name]; ok {
			itcfg := cfg.CutFromOperator(hf)
			if h := handler(itcfg); h != nil {
				opts = append(opts, h)
			}
		}
	}
	return
}

func RegisterDialOption(name string, f func(configuration *conf.Configuration) grpc.DialOption) {
	grpcDialOptions.do[name] = f
}

func RegisterUnaryClientInterceptor(name string, f func(configuration *conf.Configuration) grpc.UnaryClientInterceptor) {
	grpcDialOptions.ucit[name] = f
}

func RegisterStreamClientInterceptor(name string, f func(configuration *conf.Configuration) grpc.StreamClientInterceptor) {
	grpcDialOptions.scit[name] = f
}

func registerInternal() {
	RegisterDialOption("insecure", func(configuration *conf.Configuration) grpc.DialOption { return grpc.WithInsecure() })
	RegisterDialOption("block", func(configuration *conf.Configuration) grpc.DialOption { return grpc.WithBlock() })
}
