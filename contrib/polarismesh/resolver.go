package polarismesh

import (
	"context"
	"errors"
	"fmt"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"strings"
	"time"
)

// resolverBuilder implements the resolver.Builder interface.
type resolverBuilder struct {
	sdkCtx api.SDKContext
}

// Scheme polaris scheme
func (rb *resolverBuilder) Scheme() string {
	return scheme
}

// Build Implement the Build method in the Resolver Builder interface,
// build a new Resolver resolution service address for the specified Target,
// and pass the polaris information to the balancer through attr
func (rb *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn,
	_ resolver.BuildOptions) (resolver.Resolver, error) {
	options, err := targetToOptions(target)
	if nil != err {
		return nil, err
	}

	if options.Service == "" {
		return nil, errors.New("resolver need a target host or service name")
	}

	if rb.sdkCtx == nil {
		rb.sdkCtx, err = PolarisContext()
		if err != nil {
			return nil, err
		}
	}
	d := &polarisNamingResolver{
		consumerAPI: api.NewConsumerAPIByContext(rb.sdkCtx),
		cc:          cc,
		rn:          make(chan struct{}, 1),
		target:      target,
		options:     options,
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())

	go d.Watch()
	d.ResolveNow(resolver.ResolveNowOptions{})
	return d, nil
}

type polarisNamingResolver struct {
	ctx    context.Context
	cancel context.CancelFunc
	cc     resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn      chan struct{}
	options *dialOptions
	target  resolver.Target

	consumerAPI   api.ConsumerAPI
	watchRequest  *api.WatchAllInstancesRequest
	watcherResp   *model.WatchAllInstancesResponse
	configuration *conf.Configuration
}

// OnInstancesUpdate The method is called by the polaris client when the service instance is updated.
//
// when watch all instances, but if single instance, response cluster instances will be nil.
// so we need to get all instances again.
func (pr *polarisNamingResolver) OnInstancesUpdate(resp *model.InstancesResponse) {
	pr.onInstancesUpdate(resp, true)
}

func (pr *polarisNamingResolver) onInstancesUpdate(_ *model.InstancesResponse, skipRouteFilter bool) {
	instancesRequest := &api.GetInstancesRequest{
		GetInstancesRequest: model.GetInstancesRequest{
			Service:         pr.options.Service,
			Namespace:       getNamespace(pr.options),
			SkipRouteFilter: skipRouteFilter,
		},
	}

	resp, err := pr.consumerAPI.GetInstances(instancesRequest)
	if nil != err {
		pr.cc.ReportError(err)
		return
	}
	if skipRouteFilter {
		// filter unhealthy and isolated instances
		usedInstances := make([]model.Instance, 0, len(resp.Instances))
		totalWeight := 0
		for _, instance := range resp.Instances {
			if !instance.IsHealthy() || instance.IsIsolated() {
				continue
			}
			usedInstances = append(usedInstances, instance)
			totalWeight += instance.GetWeight()
		}
		resp.Instances = usedInstances
		resp.TotalWeight = totalWeight
	}

	state := &resolver.State{
		Attributes: attributes.New(keyDialOptions, pr.options).WithValue(keyResponse, resp),
	}
	for _, instance := range resp.Instances {
		state.Addresses = append(state.Addresses, resolver.Address{
			Addr: fmt.Sprintf("%s:%d", instance.GetHost(), instance.GetPort()),
		})
	}
	if err := pr.cc.UpdateState(*state); nil != err {
		grpclog.Errorf("fail to do update service %s: %v", pr.target.URL.String(), err)
	}
}

// ResolveNow The method is called by the gRPC framework to resolve the target name immediately.
//
// attention: this method triggers too high frequency to cause polaris server hung.
// so do not do anything until you know what you are doing.
func (pr *polarisNamingResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	select {
	case pr.rn <- struct{}{}:
	default:
	}
}

func getNamespace(options *dialOptions) string {
	namespace := DefaultNamespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}
	return namespace
}

// Watch the listener is OnInstancesUpdate
func (pr *polarisNamingResolver) Watch() {
	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-pr.rn:
			pr.OnInstancesUpdate(nil)
		}
		pr.watchRequest = &api.WatchAllInstancesRequest{
			WatchAllInstancesRequest: model.WatchAllInstancesRequest{
				ServiceKey: model.ServiceKey{
					Namespace: getNamespace(pr.options),
					Service:   pr.options.Service,
				},
				InstancesListener: pr,
				WaitTime:          time.Minute,
				WatchMode:         model.WatchModeNotify,
			},
		}

		watcher, err := pr.consumerAPI.WatchAllInstances(pr.watchRequest)
		if err != nil {
			pr.cc.ReportError(err)
			return
		}
		pr.watcherResp = watcher
	}
}

// Close resolver closed
func (pr *polarisNamingResolver) Close() {
	if pr.watcherResp != nil {
		pr.watcherResp.CancelWatch()
	}
	pr.cancel()
}

func extractBareMethodName(fullMethodName string) string {
	index := strings.LastIndex(fullMethodName, "/")
	if index == -1 {
		return ""
	}
	return fullMethodName[index+1:]
}
