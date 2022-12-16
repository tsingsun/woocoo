package polarismesh

import (
	"context"
	"github.com/polarismesh/polaris-go/api"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"path"
)

// RateLimitOptions is a gRPC interceptor that implements rate limiting.
type RateLimitOptions struct {
	Namespace string
	Service   string
	limitAPI  api.LimitAPI
}

func NewRateLimitOptions() *RateLimitOptions {
	polarisCtx, err := PolarisContext()
	if err != nil {
		panic(err)
	}
	return &RateLimitOptions{
		limitAPI: api.NewLimitAPIByContext(polarisCtx),
	}
}

func (rl *RateLimitOptions) Apply(cnf *conf.Configuration) {
	rl.Namespace = cnf.Root().Namespace()
	if err := cnf.Unmarshal(rl); err != nil {
		panic(err)
	}
}

// RateLimitUnaryServerInterceptor returns a new unary server interceptors that performs per-method rate limiting.
func RateLimitUnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	interceptor := NewRateLimitOptions()
	interceptor.Apply(cfg)
	namespace := interceptor.Namespace
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		fullMethodString := info.FullMethod
		service := path.Dir(fullMethodString)[1:]
		method := path.Base(fullMethodString)
		serviceName := service
		if interceptor.Service != "" {
			serviceName = interceptor.Service
		}
		quotaReq := api.NewQuotaRequest()
		quotaReq.SetNamespace(namespace)
		quotaReq.SetService(serviceName)
		quotaReq.SetMethod(method)
		future, err := interceptor.limitAPI.GetQuota(quotaReq)
		if err != nil {
			grpclog.Errorf("fail to do rate limit %s: %v", fullMethodString, err)
			return handler(ctx, req)
		}
		rsp := future.Get()
		if rsp.Code == api.QuotaResultLimited {
			return nil, status.Error(codes.ResourceExhausted, rsp.Info)
		}
		return handler(ctx, req)
	}
}
