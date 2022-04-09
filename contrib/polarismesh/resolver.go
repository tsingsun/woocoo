package polarismesh

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
	"sync"
)

type resolverBuilder struct {
	// for the registry
	configuration *conf.Configuration
}

// Scheme polaris scheme
func (rb *resolverBuilder) Scheme() string {
	return scheme
}

// Build Implement the Build method in the Resolver Builder interface,
// build a new Resolver resolution service address for the specified Target,
// and pass the polaris information to the balancer through attr
func (rb *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	options, err := registry.TargetToOptions(target)
	if nil != err {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &polarisNamingResolver{
		ctx:           ctx,
		cancel:        cancel,
		cc:            cc,
		rn:            make(chan struct{}, 1),
		target:        target,
		options:       options,
		configuration: rb.configuration,
	}
	d.wg.Add(1)
	go d.watcher()
	d.resolveNow()
	return d, nil
}

type polarisNamingResolver struct {
	ctx    context.Context
	cancel context.CancelFunc
	cc     resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn      chan struct{}
	wg      sync.WaitGroup
	options *registry.DialOptions
	target  resolver.Target

	configuration      *conf.Configuration
	dstMetadata        map[string]string
	hasInitDstMetadata bool
	balanceOnce        sync.Once
}

// ResolveNow The method is called by the gRPC framework to resolve the target name immediately.
//
// attention: this method trigger too high frequency to cause polaris server hung. so do not anything until you know what you are doing.
func (pr *polarisNamingResolver) ResolveNow(opt resolver.ResolveNowOptions) {
}

func (pr *polarisNamingResolver) resolveNow() {
	select {
	case pr.rn <- struct{}{}:
	default:
	}
}

func getNamespace(options *registry.DialOptions) string {
	namespace := DefaultNamespace
	if len(options.Namespace) > 0 {
		namespace = options.Namespace
	}
	return namespace
}

const keyDialOptions = "options"

func (pr *polarisNamingResolver) lookup() (*resolver.State, api.ConsumerAPI, error) {
	sdkCtx, err := PolarisContext(pr.configuration)
	if nil != err {
		return nil, nil, err
	}
	consumerAPI := api.NewConsumerAPIByContext(sdkCtx)
	instancesRequest := &api.GetInstancesRequest{}
	instancesRequest.Namespace = getNamespace(pr.options)
	instancesRequest.Service = pr.target.URL.Host

	if len(pr.dstMetadata) > 0 && pr.hasInitDstMetadata {
		instancesRequest.Metadata = pr.dstMetadata
	} else {
		instancesRequest.Metadata = filterMetadata(pr.options, "dst_")
		pr.hasInitDstMetadata = true
	}

	sourceService := buildSourceInfo(pr.options)
	if sourceService != nil {
		// 如果在Conf中配置了SourceService，则优先使用配置
		instancesRequest.SourceService = sourceService
	}
	resp, err := consumerAPI.GetInstances(instancesRequest)
	if nil != err {
		return nil, consumerAPI, err
	}
	state := &resolver.State{}
	for _, instance := range resp.Instances {
		state.Addresses = append(state.Addresses, resolver.Address{
			Addr:       fmt.Sprintf("%s:%d", instance.GetHost(), instance.GetPort()),
			Attributes: attributes.New(keyDialOptions, pr.options),
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
	defer pr.wg.Done()
	var consumerAPI api.ConsumerAPI
	var eventChan <-chan model.SubScribeEvent
	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-pr.rn:
		case ev := <-eventChan:
			ev.GetSubScribeEventType()
		}
		var state *resolver.State
		var err error
		state, consumerAPI, err = pr.lookup()
		if err != nil {
			pr.cc.ReportError(err)
		} else {
			pr.balanceOnce.Do(func() {
				state.ServiceConfig = &serviceconfig.ParseResult{
					Config: &grpc.ServiceConfig{
						LB: proto.String(scheme),
					},
				}
			})
			err = pr.cc.UpdateState(*state)
			if nil != err {
				grpclog.Errorf("fail to do update service %s: %v", pr.target.URL.Host, err)
			}
			var svcKey model.ServiceKey
			svcKey, eventChan, err = pr.doWatch(consumerAPI)
			if nil != err {
				grpclog.Errorf("fail to do watch for service %s: %v", svcKey, err)
			}
		}
	}
}

// Close resolver closed
func (pr *polarisNamingResolver) Close() {
	pr.cancel()
}
