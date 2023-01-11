package polarismesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/polarismesh/polaris-go/api"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"strings"
	"sync"
	"time"
)

const (
	scheme          = "polaris"
	headerPrefixKey = "headerPrefix"
)

func init() {
	registry.RegisterDriver(scheme, Driver{})
	balancer.Register(&balancerBuilder{})
	grpcx.RegisterGrpcUnaryInterceptor(scheme+"RateLimit", RateLimitUnaryServerInterceptor)
}

var (
	once sync.Once
	_    registry.Driver = (*Driver)(nil)
)

type (
	// Driver implementation of registry.Driver
	Driver struct {
	}
	// Options is the options for the polaris registry
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

func (drv Driver) ResolverBuilder(config *conf.Configuration) (resolver.Builder, error) {
	if err := SetPolarisConfig(config); err != nil {
		return nil, err
	}
	rb := &resolverBuilder{}
	once.Do(func() {
		balancer.Register(&balancerBuilder{})
	})
	return rb, nil
}

func (r *Registry) Register(serviceInfo *registry.ServiceInfo) error {
	info := *serviceInfo
	if serviceInfo.Metadata != nil {
		info.Metadata = make(map[string]string)
		for k, v := range serviceInfo.Metadata {
			info.Metadata[k] = v
		}
	}
	registerRequest := &api.InstanceRegisterRequest{}
	registerRequest.Namespace = info.Namespace
	registerRequest.Service = info.Name
	registerRequest.Version = &info.Version
	registerRequest.Host = info.Host
	registerRequest.Port = info.Port
	if r.opts.TTL > 0 {
		registerRequest.SetTTL(int(r.opts.TTL.Seconds()))
	}
	if info.Protocol != "" {
		registerRequest.Protocol = proto.String(info.Protocol)
	}
	// try to get the value from the config
	registerRequest.ServiceToken = info.Metadata["token"]
	delete(info.Metadata, "token")
	registerRequest.Metadata = info.Metadata
	r.registerContext.registerRequests = append(r.registerContext.registerRequests, registerRequest)
	resp, err := r.registerContext.providerAPI.RegisterInstance(registerRequest)
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

// TTL return 0, polaris use heartbeat to keep alive,so ttl is not used
func (r *Registry) TTL() time.Duration {
	return 0
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
	err := SetPolarisConfig(cfg)
	if err != nil {
		panic(err)
	}
	ctx, err := PolarisContext()
	if err != nil {
		panic(err)
	}
	r.opts.TTL = cfg.Duration("ttl")
	r.registerContext.providerAPI = api.NewProviderAPIByContext(ctx)
}

func targetToOptions(target resolver.Target) (*dialOptions, error) {
	options := &dialOptions{}
	if len(target.URL.RawQuery) > 0 {
		var optionsStr string
		values := target.URL.Query()
		if len(values) > 0 {
			optionValues := values[keyDialOptions]
			if len(optionValues) > 0 {
				optionsStr = optionValues[0]
			}
		}
		if len(optionsStr) > 0 {
			value, err := base64.URLEncoding.DecodeString(optionsStr)
			if nil != err {
				return nil, fmt.Errorf("TargetToOptions:fail to decode endpoint %s, options %s: %v",
					target.URL.Path, optionsStr, err)
			}
			ro := &registry.DialOptions{}
			if err = json.Unmarshal(value, ro); nil != err {
				return nil, fmt.Errorf("TargetToOptions:fail to unmarshal options %s: %v",
					string(value), err)
			}
			options.Namespace = ro.Namespace
			options.DstMetadata = filterMetadata(ro, "dst_")
			options.SrcMetadata = filterMetadata(ro, "src_")
			if hk, ok := ro.Metadata[headerPrefixKey]; ok {
				options.HeaderPrefix = strings.Split(hk, ",")
			}
		}
	} else {
		options.Namespace = target.URL.Host
		if target.URL.Opaque != "" {
			options.SrcService = target.URL.Opaque
		} else {
			options.SrcService = target.URL.Path
		}
	}
	return options, nil
}

// parse src metadata from registry dial options, example: src_key = value => key = value
func filterMetadata(options *registry.DialOptions, prefix string) map[string]string {
	var srcMetadata = make(map[string]string)
	for k, v := range options.Metadata {
		if strings.HasPrefix(k, prefix) {
			srcMetadata[k[len(prefix):]] = v
		}
	}
	return srcMetadata
}
