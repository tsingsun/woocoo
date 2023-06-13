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
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	scheme             = "polaris"
	keyDialOptions     = "options"
	keyDialOptionRoute = "route"
	keyResponse        = "response"
)

func init() {
	registry.RegisterDriver(scheme, &Driver{
		refRegtries: make(map[string]registry.Registry),
		refBuilders: make(map[string]resolver.Builder)},
	)
	balancer.Register(&balancerBuilder{})
	grpcx.RegisterGrpcUnaryInterceptor(scheme+"RateLimit", RateLimitUnaryServerInterceptor)
}

var (
	once sync.Once
)

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

func (drv *Driver) CreateRegistry(cnf *conf.Configuration) (registry.Registry, error) {
	drv.mu.Lock()
	defer drv.mu.Unlock()
	ccfg := cnf
	ref := cnf.String("ref")
	if ref != "" {
		if v, ok := drv.refRegtries[ref]; ok {
			return v, nil
		}
		ccfg = cnf.Root().Sub(ref)
	}
	r := New()
	r.Apply(ccfg)
	if ref != "" {
		setGlobalConfig(ccfg)
		drv.refRegtries[ref] = r
	}
	return r, nil
}

func (drv *Driver) ResolverBuilder(cnf *conf.Configuration) (resolver.Builder, error) {
	drv.mu.Lock()
	defer drv.mu.Unlock()
	ccfg := cnf
	ref := cnf.String("ref")
	if ref != "" {
		if v, ok := drv.refBuilders[ref]; ok {
			return v, nil
		}
		ccfg = cnf.Root().Sub(ref)
	}
	pc, err := NewPolarisConfig(ccfg)
	if err != nil {
		return nil, err
	}
	sdkCtx, err := api.InitContextByConfig(pc)
	if err != nil {
		return nil, err
	}

	rb := &resolverBuilder{
		config: pc,
		sdkCtx: sdkCtx,
	}
	if ref != "" {
		setGlobalConfig(ccfg)
		drv.refBuilders[ref] = rb
	}
	return rb, nil
}

// WithDialOptions returns the default dial options for the grpc Polaris GRPC Client.
func (drv *Driver) WithDialOptions(registryOpt registry.DialOptions) (opts []grpc.DialOption, err error) {
	do := &dialOptions{}
	if err = convertDialOptions(&registryOpt, do); err != nil {
		return nil, fmt.Errorf("WithDialOptions:failed to convert dial options: %v", err)
	}
	opts = append(opts, grpc.WithUnaryInterceptor(injectCallerInfo(do)), grpc.WithDefaultServiceConfig(LoadBalanceConfig))
	return
}

func setGlobalConfig(cnf *conf.Configuration) {
	once.Do(func() {
		if pc, err := NewPolarisConfig(cnf); err != nil {
			panic(err)
		} else {
			SetPolarisConfig(pc)
		}
	})
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

// Unregister  the service from the registry
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

// Apply the configuration to the registry and set the first config as the default global config
func (r *Registry) Apply(cfg *conf.Configuration) {
	pcfg, err := NewPolarisConfig(cfg)
	if err != nil {
		panic(err)
	}

	ctx, err := api.InitContextByConfig(pcfg)
	if err != nil {
		panic(err)
	}

	r.opts.TTL = cfg.Duration("ttl")
	r.registerContext.providerAPI = api.NewProviderAPIByContext(ctx)
}

// polaris parse the target to options
//
// target format :
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
				ur := values[keyDialOptionRoute]
				if len(ur) > 0 {
					options.Route, err = strconv.ParseBool(ur[0])
					if err != nil {
						return nil, fmt.Errorf("TargetToOptions:fail to parse route %s: %v", ur[0], err)
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
	tar.DstMetadata = filterMetadata(src, "dst_")
	tar.SrcMetadata = filterMetadata(src, "src_")
	if v, ok := src.Metadata["route"]; ok {
		if tar.Route, err = strconv.ParseBool(v); err != nil {
			return fmt.Errorf("metadata:route %s: %v", v, err)
		}
	}
	if v, ok := src.Metadata["src_service"]; ok {
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
