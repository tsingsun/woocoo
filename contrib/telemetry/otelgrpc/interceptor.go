package otelgrpc

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

const interceptorName = "otel"

func init() {
	grpcx.RegisterGrpcServerOption(interceptorName, func(_ *conf.Configuration) grpc.ServerOption {
		return grpc.StatsHandler(otelgrpc.NewServerHandler())
	})
	grpcx.RegisterDialOption(interceptorName, func(_ *conf.Configuration) grpc.DialOption {
		return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
	})
}
