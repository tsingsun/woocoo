package grpcx

import (
	"crypto/tls"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor/auth"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor/logger"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"path/filepath"
)

var (
	cGrpcServerOptions = newConfigurableGrpcOptions()
)

func init() {
	RegisterGrpcServerOption("keepalive", keepaliveHandler)
	RegisterGrpcServerOption("tls", tlsHandler)
	RegisterGrpcUnaryInterceptor("auth", auth.UnaryServerInterceptor)
	RegisterGrpcUnaryInterceptor("accessLog", logger.UnaryServerInterceptor)
	RegisterGrpcUnaryInterceptor("recovery", recovery.UnaryServerInterceptor)
}

type configurableGrpcServerOptions struct {
	ms   map[string]func(*conf.Configuration) grpc.ServerOption
	usit map[string]func(*conf.Configuration) grpc.UnaryServerInterceptor
	ssit map[string]func(*conf.Configuration) grpc.StreamServerInterceptor
}

func newConfigurableGrpcOptions() *configurableGrpcServerOptions {
	return &configurableGrpcServerOptions{
		ms:   make(map[string]func(*conf.Configuration) grpc.ServerOption),
		usit: make(map[string]func(*conf.Configuration) grpc.UnaryServerInterceptor),
		ssit: make(map[string]func(*conf.Configuration) grpc.StreamServerInterceptor),
	}
}

func RegisterGrpcServerOption(name string, handler func(cnf *conf.Configuration) grpc.ServerOption) error {
	cGrpcServerOptions.ms[name] = handler
	return nil
}

func RegisterGrpcUnaryInterceptor(name string, handler func(*conf.Configuration) grpc.UnaryServerInterceptor) error {
	cGrpcServerOptions.usit[name] = handler
	return nil
}

func keepaliveHandler(cfg *conf.Configuration) grpc.ServerOption {
	sp := keepalive.ServerParameters{}
	if err := cfg.Parser().UnmarshalByJson("", &sp); err != nil {
		panic(err)
	}
	return grpc.KeepaliveParams(sp)
}

func tlsHandler(cfg *conf.Configuration) grpc.ServerOption {
	ssl_certificate := cfg.String("ssl_certificate")
	ssl_certificate_key := cfg.String("ssl_certificate_key")
	if ssl_certificate != "" && ssl_certificate_key != "" {
		if !filepath.IsAbs(ssl_certificate) {
			ssl_certificate = filepath.Join(cfg.GetBaseDir(), ssl_certificate)
		}
		if !filepath.IsAbs(ssl_certificate_key) {
			ssl_certificate_key = filepath.Join(cfg.GetBaseDir(), ssl_certificate_key)
		}
		cert, err := tls.LoadX509KeyPair(ssl_certificate, ssl_certificate_key)
		if err != nil {
			panic(err)
		}
		return grpc.Creds(credentials.NewServerTLSFromCert(&cert))
	}
	return nil
}

func (c configurableGrpcServerOptions) unaryInterceptorHandler(cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.UnaryServerInterceptor
	its := cnf.SubOperator("")
	for _, it := range its {
		var name string
		for s, _ := range it.Raw() {
			name = s
			break
		}
		if handler, ok := c.usit[name]; ok {
			itcfg := cnf.CutFromOperator(it)
			opts = append(opts, handler(itcfg))
		}
	}
	return grpc.ChainUnaryInterceptor(opts...)
}

func (c configurableGrpcServerOptions) Apply(cfg *conf.Configuration, path string) (opts []grpc.ServerOption) {
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
		if handler, ok := c.ms[name]; ok {
			itcfg := cfg.CutFromOperator(hf)
			if h := handler(itcfg); h != nil {
				opts = append(opts, h)
			}
		}
	}
	return
}
