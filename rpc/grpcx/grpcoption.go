package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
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
	RegisterGrpcUnaryInterceptor("jwt", interceptor.JWTUnaryServerInterceptor)
	RegisterGrpcUnaryInterceptor("accessLog", interceptor.LoggerUnaryServerInterceptor)
	RegisterGrpcUnaryInterceptor("recovery", interceptor.RecoveryUnaryServerInterceptor)
	RegisterGrpcStreamInterceptor("jwt", interceptor.JWTSteamServerInterceptor)
	RegisterGrpcStreamInterceptor("accessLog", interceptor.LoggerStreamServerInterceptor)
	RegisterGrpcStreamInterceptor("recovery", interceptor.RecoveryStreamServerInterceptor)

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

func RegisterGrpcServerOption(name string, handler func(cnf *conf.Configuration) grpc.ServerOption) {
	cGrpcServerOptions.ms[name] = handler
}

func RegisterGrpcUnaryInterceptor(name string, handler func(*conf.Configuration) grpc.UnaryServerInterceptor) {
	cGrpcServerOptions.usit[name] = handler
}

func RegisterGrpcStreamInterceptor(name string, handler func(*conf.Configuration) grpc.StreamServerInterceptor) {
	cGrpcServerOptions.ssit[name] = handler
}

func keepaliveHandler(cfg *conf.Configuration) grpc.ServerOption {
	sp := keepalive.ServerParameters{}
	if err := cfg.Unmarshal(&sp); err != nil {
		panic(err)
	}
	return grpc.KeepaliveParams(sp)
}

func tlsHandler(cfg *conf.Configuration) grpc.ServerOption {
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
	return grpc.Creds(tc)
}

func (c configurableGrpcServerOptions) unaryInterceptorHandler(root string, cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.UnaryServerInterceptor
	cnf.Each(root, func(root string, sub *conf.Configuration) {
		if handler, ok := c.usit[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return grpc.ChainUnaryInterceptor(opts...)
}

func (c configurableGrpcServerOptions) streamServerInterceptorHandler(root string, cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.StreamServerInterceptor
	cnf.Each(root, func(root string, sub *conf.Configuration) {
		if handler, ok := c.ssit[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return grpc.ChainStreamInterceptor(opts...)
}

func (c configurableGrpcServerOptions) Apply(cnf *conf.Configuration, path string) (opts []grpc.ServerOption) {
	cnf.Each(path, func(root string, sub *conf.Configuration) {
		if root == "unaryInterceptors" {
			opts = append(opts, c.unaryInterceptorHandler(root, sub))
		} else if root == "streamInterceptors" {
			opts = append(opts, c.streamServerInterceptorHandler(root, sub))
		}
		if handler, ok := c.ms[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return
}
