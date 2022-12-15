package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/interceptor"
	"github.com/tsingsun/woocoo/rpc/grpcx/option"
	"google.golang.org/grpc"
)

var (
	optionsManager = newGrpcOptionManager()
)

func init() {
	registryIntegration()
}

type (
	grpcOptionManager struct {
		so map[string]ServerOptionFunc
		su map[string]UnaryServerInterceptorFunc
		ss map[string]StreamServerInterceptorFunc

		cd map[string]DialOptionFunc
		cu map[string]UnaryClientInterceptorFunc
		cs map[string]StreamClientInterceptorFunc

		mid map[string]interceptor.Interceptor
	}
	// server side
	ServerOptionFunc            func(*conf.Configuration) grpc.ServerOption
	UnaryServerInterceptorFunc  func(*conf.Configuration) grpc.UnaryServerInterceptor
	StreamServerInterceptorFunc func(*conf.Configuration) grpc.StreamServerInterceptor
	// client side
	DialOptionFunc              func(*conf.Configuration) grpc.DialOption
	UnaryClientInterceptorFunc  func(*conf.Configuration) grpc.UnaryClientInterceptor
	StreamClientInterceptorFunc func(*conf.Configuration) grpc.StreamClientInterceptor
)

func newGrpcOptionManager() *grpcOptionManager {
	return &grpcOptionManager{
		so: make(map[string]ServerOptionFunc),
		su: make(map[string]UnaryServerInterceptorFunc),
		ss: make(map[string]StreamServerInterceptorFunc),
		cd: make(map[string]DialOptionFunc),
		cu: make(map[string]UnaryClientInterceptorFunc),
		cs: make(map[string]StreamClientInterceptorFunc),
	}
}

func registryIntegration() {
	ka := option.KeepAliveOption{}
	tls := option.TLSOption{}
	jwt := interceptor.JWT{}
	aclog := interceptor.AccessLogger{}
	recovery := interceptor.Recovery{}
	optionsManager.so = map[string]ServerOptionFunc{
		ka.Name():  ka.ServerOption,
		tls.Name(): tls.ServerOption,
	}
	optionsManager.su = map[string]UnaryServerInterceptorFunc{
		jwt.Name():      jwt.UnaryServerInterceptor,
		aclog.Name():    aclog.UnaryServerInterceptor,
		recovery.Name(): recovery.UnaryServerInterceptor,
	}
	optionsManager.ss = map[string]StreamServerInterceptorFunc{
		jwt.Name():      jwt.SteamServerInterceptor,
		aclog.Name():    aclog.StreamServerInterceptor,
		recovery.Name(): recovery.StreamServerInterceptor,
	}
	optionsManager.cd = map[string]DialOptionFunc{
		ka.Name():  ka.DialOption,
		tls.Name(): tls.DialOption,
		"block": func(configuration *conf.Configuration) grpc.DialOption {
			return grpc.WithBlock()
		},
		"defaultServiceConfig": func(cfg *conf.Configuration) grpc.DialOption {
			return grpc.WithDefaultServiceConfig(cfg.String("defaultServiceConfig"))
		},
	}
}

// RegisterGrpcServerOption register grpc server option
func RegisterGrpcServerOption(name string, handler ServerOptionFunc) {
	optionsManager.so[name] = handler
}

// RegisterGrpcUnaryInterceptor register grpc unary interceptor
func RegisterGrpcUnaryInterceptor(name string, handler UnaryServerInterceptorFunc) {
	optionsManager.su[name] = handler
}

// RegisterGrpcStreamInterceptor register grpc stream interceptor
func RegisterGrpcStreamInterceptor(name string, handler StreamServerInterceptorFunc) {
	optionsManager.ss[name] = handler
}

// RegisterDialOption register grpc dial option on client side
func RegisterDialOption(name string, f func(configuration *conf.Configuration) grpc.DialOption) {
	optionsManager.cd[name] = f
}

// RegisterUnaryClientInterceptor register grpc unary client interceptor
func RegisterUnaryClientInterceptor(name string, f func(configuration *conf.Configuration) grpc.UnaryClientInterceptor) {
	optionsManager.cu[name] = f
}

// RegisterStreamClientInterceptor register grpc stream client interceptor
func RegisterStreamClientInterceptor(name string, f func(configuration *conf.Configuration) grpc.StreamClientInterceptor) {
	optionsManager.cs[name] = f
}
func (c grpcOptionManager) buildServerChainUnary(root string, cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.UnaryServerInterceptor
	cnf.Each(root, func(root string, sub *conf.Configuration) {
		if handler, ok := c.su[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return grpc.ChainUnaryInterceptor(opts...)
}

func (c grpcOptionManager) streamServerInterceptorHandler(root string, cnf *conf.Configuration) grpc.ServerOption {
	var opts []grpc.StreamServerInterceptor
	cnf.Each(root, func(root string, sub *conf.Configuration) {
		if handler, ok := c.ss[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return grpc.ChainStreamInterceptor(opts...)
}

// BuildServerOptions build grpc server options by config. the path node is slice type
// cnf format:
//
//	engine:
//	  - keepalive:
//	  - unaryInterceptors:
func (c grpcOptionManager) BuildServerOptions(cnf *conf.Configuration, path string) (opts []grpc.ServerOption) {
	cnf.Each(path, func(root string, sub *conf.Configuration) {
		if root == "unaryInterceptors" {
			opts = append(opts, c.buildServerChainUnary(root, sub))
		} else if root == "streamInterceptors" {
			opts = append(opts, c.streamServerInterceptorHandler(root, sub))
		}
		if optionFunc, ok := c.so[root]; ok {
			opts = append(opts, optionFunc(sub))
		}
	})
	return
}

// BuildDialOption build grpc dial option by config. the path node is slice type
func (c grpcOptionManager) BuildDialOption(client *Client, cnf *conf.Configuration, path string) (opts []grpc.DialOption) {
	cnf.Each(path, func(root string, sub *conf.Configuration) {
		if root == "timeout" {
			client.timeout = sub.Duration(root)
			return
		}
		if root == "unaryInterceptors" {
			opts = append(opts, c.buildClientChainUnary(root, sub))
		}
		if handler, ok := c.cd[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return

}

func (c grpcOptionManager) buildClientChainUnary(root string, cnf *conf.Configuration) grpc.DialOption {
	var opts []grpc.UnaryClientInterceptor
	cnf.Each(root, func(root string, sub *conf.Configuration) {
		if handler, ok := c.cu[root]; ok {
			opts = append(opts, handler(sub))
		}
	})
	return grpc.WithChainUnaryInterceptor(opts...)
}
