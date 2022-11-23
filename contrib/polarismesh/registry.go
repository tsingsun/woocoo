package polarismesh

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/polarismesh/polaris-go/api"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

const scheme = "polaris"

func init() {
	registry.RegisterDriver(scheme, Driver{})
	balancer.Register(&balancerBuilder{})
}

var (
	once sync.Once
)

type (
	Driver struct {
	}

	Options struct {
		TTL time.Duration `json:"ttl" yaml:"ttl"`
	}

	RegisterContext struct {
		providerAPI      api.ProviderAPI
		registerRequests []*api.InstanceRegisterRequest
		cancel           context.CancelFunc
		healthCheckWait  *sync.WaitGroup
	}

	Registry struct {
		opts            Options
		registerContext *RegisterContext
	}
)

func (drv Driver) CreateRegistry(config *conf.Configuration) (registry.Registry, error) {
	r := New()
	r.Apply(config)
	return r, nil
}

func (r *Registry) ResolverBuilder(config *conf.Configuration) resolver.Builder {
	rb := &resolverBuilder{
		configuration: config,
	}
	once.Do(func() {
		balancer.Register(&balancerBuilder{})
	})
	return rb
}

func (drv Driver) ResolverBuilder(config *conf.Configuration) (resolver.Builder, error) {
	rb := &resolverBuilder{
		configuration: config,
	}
	once.Do(func() {
		balancer.Register(&balancerBuilder{})
	})
	return rb, nil
}

func (r *Registry) Register(serviceInfo *registry.ServiceInfo) error {
	registerRequest := &api.InstanceRegisterRequest{}
	registerRequest.Namespace = serviceInfo.Namespace
	registerRequest.Service = serviceInfo.Name
	//registerRequest.Version = &serviceInfo.Version
	registerRequest.Host = serviceInfo.Host
	registerRequest.Port = serviceInfo.Port
	registerRequest.SetTTL(int(r.opts.TTL.Seconds()))
	registerRequest.Protocol = proto.String(serviceInfo.Protocol)
	// try to get the value from the config
	registerRequest.ServiceToken = serviceInfo.Metadata["token"]
	delete(serviceInfo.Metadata, "token")
	registerRequest.Metadata = serviceInfo.Metadata
	r.registerContext.registerRequests = append(r.registerContext.registerRequests, registerRequest)
	resp, err := r.registerContext.providerAPI.Register(registerRequest)
	if err != nil {
		return err
	}
	grpclog.Infof("[Polaris]success to register %s:%d to service %s(%s), id %s",
		registerRequest.Host, registerRequest.Port, registerRequest.Service, registerRequest.Namespace, resp.InstanceID)
	return nil
}

// Unregister  the service from the registry
// if the service is not registered, return nil
func (r *Registry) Unregister(serviceInfo *registry.ServiceInfo) error {
	deregisterRequest := &api.InstanceDeRegisterRequest{}
	deregisterRequest.Namespace = serviceInfo.Namespace
	deregisterRequest.Service = serviceInfo.Name
	deregisterRequest.Host = serviceInfo.Host
	deregisterRequest.Port = serviceInfo.Port
	deregisterRequest.ServiceToken = serviceInfo.Metadata["token"]
	err := r.registerContext.providerAPI.Deregister(deregisterRequest)
	if nil != err {
		return err
	}
	return nil
}

func (r *Registry) TTL() time.Duration {
	return r.opts.TTL
}

func (r *Registry) Close() {
	r.registerContext.cancel()
}

func New() *Registry {
	_, cancel := context.WithCancel(context.Background())
	registerContext := &RegisterContext{
		cancel: cancel,
	}
	r := &Registry{
		opts:            Options{},
		registerContext: registerContext,
	}
	return r
}

func (r *Registry) Apply(cfg *conf.Configuration) {
	ctx, err := PolarisContext(cfg)
	if nil != err {
		panic(err)
	}
	r.opts.TTL = cfg.Duration("ttl")
	r.registerContext.providerAPI = api.NewProviderAPIByContext(ctx)
}
