package otelgrpc

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/client"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

const interceptorName = "otel"

func init() {
	grpcx.RegisterGrpcUnaryInterceptor(interceptorName, UnaryServerInterceptor)
	grpcx.RegisterGrpcStreamInterceptor(interceptorName, StreamServerInterceptor)
	client.RegisterUnaryClientInterceptor(interceptorName, UnaryClientInterceptor)
	client.RegisterStreamClientInterceptor(interceptorName, StreamClientInterceptor)
}

func UnaryServerInterceptor(_ *conf.Configuration) grpc.UnaryServerInterceptor {
	return otelgrpc.UnaryServerInterceptor()
}

func StreamServerInterceptor(_ *conf.Configuration) grpc.StreamServerInterceptor {
	return otelgrpc.StreamServerInterceptor()
}

func UnaryClientInterceptor(_ *conf.Configuration) grpc.UnaryClientInterceptor {
	return otelgrpc.UnaryClientInterceptor()
}

func StreamClientInterceptor(_ *conf.Configuration) grpc.StreamClientInterceptor {
	return otelgrpc.StreamClientInterceptor()
}
