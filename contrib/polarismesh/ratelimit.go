package polarismesh

import (
	"context"
	"fmt"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/flow/data"
	"github.com/polarismesh/polaris-go/pkg/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

// RateLimitInterceptor is a gRPC interceptor that implements rate limiting.
type RateLimitInterceptor struct {
	Namespace string
	Service   string
	limitAPI  api.LimitAPI
}

func NewRateLimitOptions() *RateLimitInterceptor {
	polarisCtx, err := PolarisContext()
	if err != nil {
		panic(err)
	}
	return &RateLimitInterceptor{
		limitAPI: api.NewLimitAPIByContext(polarisCtx),
	}
}

func (rl *RateLimitInterceptor) Apply(cnf *conf.Configuration) {
	rl.Namespace = cnf.Root().Namespace()
	if err := cnf.Unmarshal(rl); err != nil {
		panic(err)
	}
}

func (rl *RateLimitInterceptor) buildQuotaRequest(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo) api.QuotaRequest {

	fullMethodName := info.FullMethod
	tokens := strings.Split(fullMethodName, "/")
	if len(tokens) != 3 {
		return nil
	}
	namespace := rl.Namespace

	quotaReq := api.NewQuotaRequest()
	quotaReq.SetNamespace(namespace)
	quotaReq.SetService(tokens[1])
	quotaReq.SetMethod(tokens[2])

	if len(rl.Service) > 0 {
		quotaReq.SetService(rl.Service)
		quotaReq.SetMethod(fullMethodName)
	}

	matchs, ok := rl.fetchArguments(quotaReq.(*model.QuotaRequestImpl))
	if !ok {
		return quotaReq
	}
	header, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		header = metadata.MD{}
	}

	for i := range matchs {
		item := matchs[i]
		switch item.GetType() {
		case apitraffic.MatchArgument_CALLER_SERVICE:
			serviceValues := header.Get(polarisCallerServiceKey)
			namespaceValues := header.Get(polarisCallerNamespaceKey)
			if len(serviceValues) > 0 && len(namespaceValues) > 0 {
				quotaReq.AddArgument(model.BuildCallerServiceArgument(namespaceValues[0], serviceValues[0]))
			}
		case apitraffic.MatchArgument_HEADER:
			values := header.Get(item.GetKey())
			if len(values) > 0 {
				quotaReq.AddArgument(model.BuildHeaderArgument(item.GetKey(), fmt.Sprintf("%+v", values[0])))
			}
		case apitraffic.MatchArgument_CALLER_IP:
			if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
				address := pr.Addr.String()
				addrSlice := strings.Split(address, ":")
				if len(addrSlice) == 2 {
					clientIP := addrSlice[0]
					quotaReq.AddArgument(model.BuildCallerIPArgument(clientIP))
				}
			}
		}
	}

	return quotaReq
}

func (rl *RateLimitInterceptor) fetchArguments(req *model.QuotaRequestImpl) ([]*apitraffic.MatchArgument, bool) {
	engine := rl.limitAPI.SDKContext().GetEngine()

	getRuleReq := &data.CommonRateLimitRequest{
		DstService: model.ServiceKey{
			Namespace: req.GetNamespace(),
			Service:   req.GetService(),
		},
		Trigger: model.NotifyTrigger{
			EnableDstRateLimit: true,
		},
		ControlParam: model.ControlParam{
			Timeout: time.Millisecond * 500,
		},
	}

	if err := engine.SyncGetResources(getRuleReq); err != nil {
		grpclog.Errorf("[Polaris][RateLimit] ns:%s svc:%s get RateLimit Rule fail : %+v",
			req.GetNamespace(), req.GetService(), err)
		return nil, false
	}

	svcRule := getRuleReq.RateLimitRule
	if svcRule == nil || svcRule.GetValue() == nil {
		grpclog.Warningf("[Polaris][RateLimit] ns:%s svc:%s get RateLimit Rule is nil",
			req.GetNamespace(), req.GetService())
		return nil, false
	}

	rules, ok := svcRule.GetValue().(*apitraffic.RateLimit)
	if !ok {
		grpclog.Errorf("[Polaris][RateLimit] ns:%s svc:%s get RateLimit Rule invalid",
			req.GetNamespace(), req.GetService())
		return nil, false
	}

	ret := make([]*apitraffic.MatchArgument, 0, 4)
	for i := range rules.GetRules() {
		rule := rules.GetRules()[i]
		if len(rule.GetArguments()) == 0 {
			continue
		}

		ret = append(ret, rule.Arguments...)
	}
	return ret, true
}

// RateLimitUnaryServerInterceptor returns a new unary server interceptors that performs per-method rate limiting.
func RateLimitUnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	interceptor := NewRateLimitOptions()
	interceptor.Apply(cfg)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		quotaReq := interceptor.buildQuotaRequest(ctx, req, info)
		if quotaReq == nil {
			return handler(ctx, req)
		}
		future, err := interceptor.limitAPI.GetQuota(quotaReq)
		if err != nil {
			grpclog.Errorf("[Polaris][RateLimit] fail to get quota %#v: %v", quotaReq, err)
			return handler(ctx, req)
		}

		if rsp := future.Get(); rsp.Code == api.QuotaResultLimited {
			return nil, status.Error(codes.ResourceExhausted, rsp.Info)
		}
		return handler(ctx, req)
	}
}
