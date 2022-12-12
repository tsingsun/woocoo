package client

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"path/filepath"
)

var (
	grpcDialOptions = newConfigurableGrpcDialOptions()
)

func init() {
	registerInternal()
}

type Option func(client *Client)

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

func (c configurableGrpcClientOptions) unaryInterceptorHandler(root string, cnf *conf.Configuration) grpc.DialOption {
	var opts []grpc.UnaryClientInterceptor
	cnf.Each(root, func(root string, sub *conf.Configuration) {
		if handler, ok := c.ucit[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return grpc.WithChainUnaryInterceptor(opts...)
}

func (c configurableGrpcClientOptions) Apply(client *Client, cfg *conf.Configuration, path string) (opts []grpc.DialOption) {
	cfg.Each(path, func(root string, sub *conf.Configuration) {
		if root == "timeout" {
			client.timeout = sub.Duration(root)
			return
		}
		if root == "unaryInterceptors" {
			opts = append(opts, c.unaryInterceptorHandler(root, sub))
		}
		if handler, ok := c.do[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
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
	RegisterDialOption("insecure", func(cfg *conf.Configuration) grpc.DialOption {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	})
	RegisterDialOption("tls", func(cfg *conf.Configuration) grpc.DialOption {
		sslCertificate := cfg.String("sslCertificate")
		sslCertificateKey := cfg.String("sslCertificateKey")
		if !filepath.IsAbs(sslCertificate) {
			sslCertificate = filepath.Join(cfg.GetBaseDir(), sslCertificate)
		}
		if !filepath.IsAbs(sslCertificateKey) {
			sslCertificateKey = filepath.Join(cfg.GetBaseDir(), sslCertificateKey)
		}
		tc, err := credentials.NewServerTLSFromFile(sslCertificate, sslCertificateKey)
		if err != nil {
			panic(err)
		}
		return grpc.WithTransportCredentials(tc)
	})
	RegisterDialOption("block", func(cfg *conf.Configuration) grpc.DialOption { return grpc.WithBlock() })
	RegisterDialOption("defaultServiceConfig", func(cfg *conf.Configuration) grpc.DialOption {
		return grpc.WithDefaultServiceConfig(cfg.String("defaultServiceConfig"))
	})
}
