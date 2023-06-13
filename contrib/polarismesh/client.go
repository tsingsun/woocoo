package polarismesh

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// dialOptions is the options for attach caller info
type dialOptions struct {
	Namespace   string            `yaml:"namespace" json:"namespace"`
	Service     string            `yaml:"service" json:"service"`
	DstMetadata map[string]string `yaml:"dst_metadata" json:"dst_metadata"`
	// SrcMetadata will be added to the outgoing context
	SrcMetadata map[string]string `yaml:"src_metadata" json:"src_metadata"`
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
		for h, v := range options.SrcMetadata {
			ctx = metadata.AppendToOutgoingContext(ctx, h, v)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
