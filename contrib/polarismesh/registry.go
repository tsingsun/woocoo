package polarismesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"google.golang.org/protobuf/proto"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	scheme                      = "polaris"
	globalConfigKey             = "global"
	keyDialOptions              = "options"
	keyDialOptionRoute          = "route"
	keyDialOptionNamespace      = "namespace"
	keyDialOptionService        = "service"
	keyDialOptionSrcService     = "srcService"
	keyDialOptionCircuitBreaker = "circuitBreaker"
	keyResponse                 = "response"
)

func init() {
	registry.RegisterDriver(scheme, &Driver{
		refRegtries: make(map[string]registry.Registry),
		refBuilders: make(map[string]resolver.Builder)},
	)
	// register default polaris balancer
	balancer.Register(NewBalancerBuilder(scheme, nil))

	grpcx.RegisterGrpcUnaryInterceptor(scheme+"RateLimit", RateLimitUnaryServerInterceptor)
}

type (
	// Driver implementation of registry.Driver. It is used to create a polaris registry
	Driver struct {
		refRegtries map[string]registry.Registry
		refBuilders map[string]resolver.Builder
		mu          sync.RWMutex
	}
	// Options is the options for the polaris registry
	Options struct {
		// TTL must between (0s, 60s) see: polaris.yaml
		TTL time.Duration `json:"ttl" yaml:"ttl"`
	}

	RegisterContext struct {
		sdkctx           api.SDKContext
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

func (drv *Driver) GetRegistry(name string) (registry.Registry, error) {
	r, ok := drv.refRegtries[name]
	if !ok {
		return nil, errors.New("registry not found,may not set it a reference registry")
	}
	return r, nil
}

// CreateRegistry creates a polaris registry.
func (drv *Driver) CreateRegistry(cfg *conf.Configuration) (registry.Registry, error) {
	drv.mu.Lock()
	defer drv.mu.Unlock()

	name := cfg.String("name")
	if name == "" {
		name = scheme
	}
	if v, ok := drv.refRegtries[name]; ok {
		return v, nil
	}

	r := New()
	r.Apply(cfg)

	if !cfg.Bool(globalConfigKey) {
		drv.refRegtries[name] = r
	}
	return r, nil
}

// ResolverBuilder creates a polaris resolver builder.
func (drv *Driver) ResolverBuilder(cfg *conf.Configuration) (resolver.Builder, error) {
	drv.mu.Lock()
	defer drv.mu.Unlock()
	var (
		rb = &resolverBuilder{}
	)
	name := cfg.String("name")
	if name == "" {
		name = scheme
	}
	if v, ok := drv.refBuilders[name]; ok {
		return v, nil
	}
	sdkCtx, err := InitPolarisContext(cfg)
	if err != nil {
		return nil, err
	}
	rb.sdkCtx = sdkCtx
	if !cfg.Bool(globalConfigKey) {
		drv.refBuilders[name] = rb
	}
	return rb, nil
}

// WithDialOptions returns the default dial options for the grpc Polaris GRPC Client.
func (drv *Driver) WithDialOptions(registryOpt registry.DialOptions) (opts []grpc.DialOption, err error) {
	do := &dialOptions{}
	if err = convertDialOptions(&registryOpt, do); err != nil {
		return nil, fmt.Errorf("WithDialOptions:failed to convert dial options: %v", err)
	}
	opts = append(opts,
		grpc.WithChainUnaryInterceptor(injectCallerInfo(do)),
		grpc.WithDefaultServiceConfig(LoadBalanceConfig),
	)
	return
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
	registerRequest.ServiceToken = info.Metadata[registerServiceTokenKey]
	delete(info.Metadata, registerServiceTokenKey)
	registerRequest.Metadata = info.Metadata
	r.registerContext.registerRequests = append(r.registerContext.registerRequests, registerRequest)
	resp, err := r.registerContext.providerAPI.RegisterInstance(registerRequest)
	if err != nil {
		return err
	}
	grpclog.Infof("[Polaris][Naming]success to register %s:%d to service %s(%s), id %s",
		registerRequest.Host, registerRequest.Port, registerRequest.Service, registerRequest.Namespace, resp.InstanceID)
	return nil
}

// Unregister the service from the registry
// if the service is not registered, return nil
func (r *Registry) Unregister(serviceInfo *registry.ServiceInfo) error {
	deregisterRequest := &api.InstanceDeRegisterRequest{}
	deregisterRequest.Namespace = serviceInfo.Namespace
	deregisterRequest.Service = serviceInfo.Name
	deregisterRequest.Host = serviceInfo.Host
	deregisterRequest.Port = serviceInfo.Port
	deregisterRequest.ServiceToken = serviceInfo.Metadata[registerServiceTokenKey]
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

// GetServiceInfos implements the registry interface
func (r *Registry) GetServiceInfos(service string) ([]*registry.ServiceInfo, error) {
	namespace := DefaultNamespace
	parts := strings.Split(service, "/")
	if lp := len(parts); lp > 1 {
		service = parts[lp-1]
		namespace = strings.Join(parts[:lp-1], "/")
	}
	consumerAPI := polaris.NewConsumerAPIByContext(r.registerContext.sdkctx)
	instancesRequest := &polaris.GetInstancesRequest{
		GetInstancesRequest: model.GetInstancesRequest{
			Service:         service,
			Namespace:       namespace,
			SkipRouteFilter: true,
		},
	}

	resp, err := consumerAPI.GetInstances(instancesRequest)
	if err != nil {
		return nil, err
	}
	var infos = make([]*registry.ServiceInfo, 0, len(resp.Instances))
	for _, instance := range resp.Instances {
		if !instance.IsHealthy() || instance.IsIsolated() {
			continue
		}
		infos = append(infos, &registry.ServiceInfo{
			Namespace: instance.GetNamespace(),
			Name:      instance.GetService(),
			Host:      instance.GetHost(),
			Port:      int(instance.GetPort()),
			Version:   instance.GetVersion(),
			Protocol:  instance.GetProtocol(),
			Metadata:  instance.GetMetadata(),
		})
	}
	return infos, nil
}

// Apply the configuration to the registry and set the first config as the default global config
func (r *Registry) Apply(cnf *conf.Configuration) {
	r.opts.TTL = cnf.Duration("ttl")
	ctx, err := InitPolarisContext(cnf)
	if err != nil {
		panic(err)
	}
	r.registerContext.sdkctx = ctx
	r.registerContext.providerAPI = api.NewProviderAPIByContext(ctx)
}

// get polaris context from registry driver,the ref name is polaris registry config ref name.
func getPolarisContextFromDriver(schemeName string) (ctx api.SDKContext, err error) {
	rd, ok := registry.GetRegistry(scheme)
	if !ok {
		return nil, fmt.Errorf("getRegistryContext: registry driver not found, scheme: %s", scheme)
	}
	drv, ok := rd.(*Driver)
	if !ok {
		return nil, fmt.Errorf("getRegistryContext: registry driver not found, scheme: %s", scheme)
	}
	r, err := drv.GetRegistry(schemeName)
	if err != nil {
		return nil, err
	}
	tr := r.(*Registry)
	return tr.registerContext.sdkctx, nil
}

// polaris parse the target to options
//
// target format:
//  1. polaris://<namespace>/<service>?key1=value1&key2=value2
//  2. polaris://<service>?<options=<jsonstr>>
func targetToOptions(target resolver.Target) (options *dialOptions, err error) {
	options = &dialOptions{}
	if target.URL.Path != "" { // no service name,parse from raw query
		options.Namespace = target.URL.Host
		if target.URL.Opaque != "" {
			options.Service = strings.TrimPrefix(target.URL.Opaque, "/")
		} else {
			options.Service = strings.TrimPrefix(target.URL.Path, "/")
		}
	}
	if len(target.URL.RawQuery) > 0 {
		var optionsStr string
		values := target.URL.Query()
		if len(values) > 0 {
			optionValues := values[keyDialOptions]
			if len(optionValues) > 0 {
				optionsStr = optionValues[0]
			} else {
				options.SrcMetadata = make(map[string]string)
				for k, v := range values {
					switch strings.ToLower(k) {
					case keyDialOptionNamespace:
						options.Namespace = v[0]
					case keyDialOptionService:
						options.Service = v[0]
					case strings.ToLower(keyDialOptionSrcService):
						options.SrcService = v[0]
					case keyDialOptionRoute:
						if len(v) > 0 {
							options.Route, err = strconv.ParseBool(v[0])
							if err != nil {
								return nil, fmt.Errorf("TargetToOptions:fail to parse route %s: %v", v[0], err)
							}
						}
					default:
						if strings.HasPrefix(k, "src_") {
							options.SrcMetadata[k[4:]] = v[0]
						}
					}
				}
			}
		}
		// parse options from query
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
			if err = convertDialOptions(ro, options); err != nil {
				return nil, fmt.Errorf("TargetToOptions:fail to parse target: %v", err)
			}
		}
	}
	return options, nil
}

func convertDialOptions(src *registry.DialOptions, tar *dialOptions) (err error) {
	tar.Namespace = src.Namespace
	tar.Service = src.ServiceName
	tar.Headers = filterMetadata(src, "header_")
	tar.SrcMetadata = filterMetadata(src, "src_")
	if v, ok := src.Metadata[keyDialOptionRoute]; ok {
		if tar.Route, err = strconv.ParseBool(v); err != nil {
			return fmt.Errorf("metadata:route %s: %v", v, err)
		}
	}
	if v, ok := src.Metadata[keyDialOptionCircuitBreaker]; ok {
		if tar.CircuitBreaker, err = strconv.ParseBool(v); err != nil {
			return fmt.Errorf("metadata:circuit breaker %s: %v", v, err)
		}
	}
	if v, ok := src.Metadata[keyDialOptionSrcService]; ok {
		tar.SrcService = v
	}
	return nil
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
