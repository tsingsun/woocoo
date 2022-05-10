package grpcx

import (
	"crypto/tls"
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
	_ = RegisterGrpcServerOption("keepalive", keepaliveHandler)
	_ = RegisterGrpcServerOption("tls", tlsHandler)
	_ = RegisterGrpcUnaryInterceptor("jwt", interceptor.JWTUnaryServerInterceptor)
	_ = RegisterGrpcUnaryInterceptor("accessLog", interceptor.LoggerUnaryServerInterceptor)
	_ = RegisterGrpcUnaryInterceptor("recovery", interceptor.RecoveryUnaryServerInterceptor)
	_ = RegisterGrpcStreamInterceptor("jwt", interceptor.JWTSteamServerInterceptor)
	_ = RegisterGrpcStreamInterceptor("accessLog", interceptor.LoggerStreamServerInterceptor)
	_ = RegisterGrpcStreamInterceptor("recovery", interceptor.RecoveryStreamServerInterceptor)

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

func RegisterGrpcStreamInterceptor(name string, handler func(*conf.Configuration) grpc.StreamServerInterceptor) error {
	cGrpcServerOptions.ssit[name] = handler
	return nil
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
	if sslCertificate != "" && sslCertificateKey != "" {
		if !filepath.IsAbs(sslCertificate) {
			sslCertificate = filepath.Join(cfg.GetBaseDir(), sslCertificate)
		}
		if !filepath.IsAbs(sslCertificateKey) {
			sslCertificateKey = filepath.Join(cfg.GetBaseDir(), sslCertificateKey)
		}
		cert, err := tls.LoadX509KeyPair(sslCertificate, sslCertificateKey)
		if err != nil {
			panic(err)
		}
		return grpc.Creds(credentials.NewServerTLSFromCert(&cert))
	}
	return nil
}

func (c configurableGrpcServerOptions) unaryInterceptorHandler(cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.UnaryServerInterceptor
	its, err := cnf.SubOperator("")
	if err != nil {
		panic(err)
	}
	for _, it := range its {
		var name string
		for s := range it.Raw() {
			name = s
			break
		}
		if handler, ok := c.usit[name]; ok {
			itcfg := cnf.CutFromOperator(it.Cut(name))
			opts = append(opts, handler(itcfg))
		}
	}
	return grpc.ChainUnaryInterceptor(opts...)
}

func (c configurableGrpcServerOptions) streamServerInterceptorHandler(cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.StreamServerInterceptor
	its, err := cnf.SubOperator("")
	if err != nil {
		panic(err)
	}
	for _, it := range its {
		var name string
		for s := range it.Raw() {
			name = s
			break
		}
		if handler, ok := c.ssit[name]; ok {
			itcfg := cnf.CutFromOperator(it.Cut(name))
			opts = append(opts, handler(itcfg))
		}
	}
	return grpc.ChainStreamInterceptor(opts...)
}

func (c configurableGrpcServerOptions) Apply(cfg *conf.Configuration, path string) (opts []grpc.ServerOption) {
	hfs := cfg.ParserOperator().Slices(path)
	for _, hf := range hfs {
		var name string
		for s := range hf.Raw() {
			name = s
			break
		}
		if name == "unaryInterceptors" {
			itcfg := cfg.CutFromOperator(hf)
			opts = append(opts, c.unaryInterceptorHandler(itcfg))
		} else if name == "streamInterceptors" {
			itcfg := cfg.CutFromOperator(hf)
			opts = append(opts, c.streamServerInterceptorHandler(itcfg))
		}
		if handler, ok := c.ms[name]; ok {
			subhf := hf.Cut(name)
			// if subhf is empty,pass the original config
			if len(subhf.Keys()) == 0 {
				subhf = hf
			}
			if h := handler(cfg.CutFromOperator(subhf)); h != nil {
				opts = append(opts, h)
			}
		}
	}
	return
}
