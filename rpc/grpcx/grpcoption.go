package grpcx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"strings"
)

var (
	cGrpcServerOptions = newConfigurableGrpcOptions()
	slist              = []string{"keepalive"}
)

func init() {
	RegisterGrpcServerOptions("keepalive", keepaliveHandler)
}

type configurableGrpcServerOptions struct {
	ms   map[string]func(*conf.Configuration, string) grpc.ServerOption
	usit map[string]func(*conf.Configuration, string) grpc.UnaryServerInterceptor
	ssit map[string]func(*conf.Configuration, string) grpc.StreamServerInterceptor
}

func newConfigurableGrpcOptions() *configurableGrpcServerOptions {
	return &configurableGrpcServerOptions{
		ms:   make(map[string]func(cnf *conf.Configuration, path string) grpc.ServerOption),
		usit: make(map[string]func(*conf.Configuration, string) grpc.UnaryServerInterceptor),
		ssit: make(map[string]func(*conf.Configuration, string) grpc.StreamServerInterceptor),
	}
}

func RegisterGrpcServerOptions(name string, handler func(cnf *conf.Configuration, path string) grpc.ServerOption) error {
	cGrpcServerOptions.ms[name] = handler
	return nil
}

func RegisterGrpcUnaryInterceptor(name string, hanlder func(*conf.Configuration, string) grpc.UnaryServerInterceptor) error {
	cGrpcServerOptions.usit[name] = hanlder
	return nil
}

func keepaliveHandler(cnf *conf.Configuration, path string) grpc.ServerOption {
	sp := keepalive.ServerParameters{}
	if err := cnf.Parser().UnmarshalByJson(path, &sp); err != nil {
		panic(err)
	}
	return grpc.KeepaliveParams(sp)
}

func unaryInterceptorHandler(cnf *conf.Configuration, path string) grpc.ServerOption {
	var opts []grpc.UnaryServerInterceptor
	its := cnf.ParserOperator().Slices(path)
	for _, it := range its {
		var name string
		for s, _ := range it.Raw() {
			name = s
			break
		}
		if handler, ok := cGrpcServerOptions.usit[name]; ok {
			k := strings.Join([]string{path, name}, conf.KeyDelimiter)
			opts = append(opts, handler(cnf, k))
		}
	}
	return grpc.ChainUnaryInterceptor(opts...)
}

func accessLogInt(cnf *conf.Configuration, path string) grpc.UnaryServerInterceptor {

	return nil
}

func (c configurableGrpcServerOptions) Apply(cnf *conf.Configuration, path string) (opts []grpc.ServerOption) {
	hfs := cnf.ParserOperator().Slices(path)
	for _, hf := range hfs {
		var name string
		for s, _ := range hf.Raw() {
			name = s
			break
		}
		if handler, ok := cGrpcServerOptions.ms[name]; ok {
			k := strings.Join([]string{path, name}, conf.KeyDelimiter)
			opts = append(opts, handler(cnf, k))
		}
	}
	return
}
