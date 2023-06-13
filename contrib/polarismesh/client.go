package polarismesh

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// dialOptions is the options for attach caller info
type dialOptions struct {
	Namespace string `yaml:"namespace" json:"namespace"`
	Service   string `yaml:"service" json:"service"`
	// Headers will be added to the outgoing context,are fixed values,parse from meatedata
	Headers map[string]string `yaml:"-" json:"-"`
	// the service name of the caller
	SrcService     string `yaml:"srcService" json:"srcService"`
	Route          bool   `yaml:"route" json:"route"`
	CircuitBreaker bool   `yaml:"circuitBreaker" json:"circuitBreaker"`
}

func injectCallerInfo(options *dialOptions) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		if _, ok := metadata.FromOutgoingContext(ctx); !ok {
			ctx = metadata.NewOutgoingContext(context.Background(), metadata.MD{})
		}

		if options.SrcService != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, polarisCallerServiceKey, options.SrcService)
			ctx = metadata.AppendToOutgoingContext(ctx, polarisCallerNamespaceKey, options.Namespace)
		}

		for k, v := range options.Headers {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		if status.Code(err) == codes.ResourceExhausted {
			return err
		}
		return err
	}
}
