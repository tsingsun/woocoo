package polarismesh

import (
	"context"
	"errors"
	"fmt"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"strconv"
	"strings"
	"sync"
	"time"
)

// resolverBuilder implements the resolver.Builder interface.
type resolverBuilder struct {
	// the polaris client config
	config config.Configuration
	sdkCtx api.SDKContext
}

// Scheme polaris scheme
func (rb *resolverBuilder) Scheme() string {
	return scheme
}

// Build Implement the Build method in the Resolver Builder interface,
// build a new Resolver resolution service address for the specified Target,
// and pass the polaris information to the balancer through attr
func (rb *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	options, err := targetToOptions(target)
	if nil != err {
		return nil, err
	}

	if rb.config == nil {
		rb.config = PolarisConfig()
		rb.sdkCtx, err = PolarisContext()
	}
	d := &polarisNamingResolver{
		sdkCtx:  rb.sdkCtx,
		cc:      cc,
		rn:      make(chan struct{}, 1),
		target:  target,
		options: options,
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())

	d.wg.Add(1)
	go d.watcher()
	d.ResolveNow(resolver.ResolveNowOptions{})
	return d, nil
}

func parseHost(target string) (string, int, error) {
	splits := strings.Split(target, ":")
	if len(splits) > 2 {
		return "", 0, errors.New("error format host")
	}
	if len(splits) == 1 {
		return target, 0, nil
	}
	port, err := strconv.Atoi(splits[1])
	if err != nil {
		return "", 0, err
	}
	return splits[0], port, nil
}

type polarisNamingResolver struct {
	sdkCtx api.SDKContext
	ctx    context.Context
	cancel context.CancelFunc
	cc     resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn      chan struct{}
	wg      sync.WaitGroup
	options *dialOptions
	target  resolver.Target

	configuration *conf.Configuration
	dstMetadata   map[string]string
}

// ResolveNow The method is called by the gRPC framework to resolve the target name immediately.
//
// attention: this method trigger too high frequency to cause polaris server hung. so do not anything until you know what you are doing.
func (pr *polarisNamingResolver) ResolveNow(opt resolver.ResolveNowOptions) {
	select {
	case pr.rn <- struct{}{}:
	default:
	}
}

func getNamespace(options *dialOptions) string {
	namespace := DefaultNamespace
	if len(options.Namespace) > 0 {
		namespace = options.Namespace
	}
	return namespace
}

func (pr *polarisNamingResolver) lookup() (*resolver.State, api.ConsumerAPI, error) {
	consumerAPI := api.NewConsumerAPIByContext(pr.sdkCtx)
	instancesRequest := &api.GetInstancesRequest{
		GetInstancesRequest: model.GetInstancesRequest{
			Service:         pr.options.SrcService,
			Namespace:       getNamespace(pr.options),
			SkipRouteFilter: true,
		},
	}
	if len(pr.options.DstMetadata) > 0 {
		instancesRequest.Metadata = pr.options.DstMetadata
	}

	resp, err := consumerAPI.GetInstances(instancesRequest)
	if nil != err {
		return nil, consumerAPI, err
	}

	updated := false
	for _, instance := range resp.Instances {
		if !instance.IsHealthy() || instance.IsIsolated() { // 过滤掉不健康和隔离的。
			updated = true
			break
		}
	}
	if updated { // 少数情况，避免创建 slice
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
	return state, consumerAPI, nil
}

func (pr *polarisNamingResolver) doWatch(
	consumerAPI api.ConsumerAPI) (model.ServiceKey, <-chan model.SubScribeEvent, error) {
	watchRequest := &api.WatchServiceRequest{}
	watchRequest.Key = model.ServiceKey{
		Namespace: getNamespace(pr.options),
		Service:   pr.target.URL.Host,
	}
	resp, err := consumerAPI.WatchService(watchRequest)
	if nil != err {
		return watchRequest.Key, nil, err
	}
	return watchRequest.Key, resp.EventChannel, nil
}

func (pr *polarisNamingResolver) watcher() {
	var (
		consumerAPI api.ConsumerAPI
		eventChan   <-chan model.SubScribeEvent
	)
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		pr.wg.Done()
	}()
	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-pr.rn:
		case <-eventChan:
		case <-ticker.C:
		}
		var (
			state *resolver.State
			err   error
		)
		state, consumerAPI, err = pr.lookup()
		if err != nil {
			pr.cc.ReportError(err)
			continue
		}
		if err = pr.cc.UpdateState(*state); nil != err {
			grpclog.Errorf("fail to do update service %s: %v", pr.target.URL, err)
		}
		var svcKey model.ServiceKey
		svcKey, eventChan, err = pr.doWatch(consumerAPI)
		if nil != err {
			grpclog.Errorf("fail to do watch for service %s: %v", svcKey, err)
		}
	}
}

// Close resolver closed
func (pr *polarisNamingResolver) Close() {
	pr.cancel()
}
