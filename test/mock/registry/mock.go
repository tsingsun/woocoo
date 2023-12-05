package registry

import (
	"errors"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

var (
	scheme = "mock"
)

type Driver struct {
	cache map[string]registry.Registry
	mu    sync.RWMutex
}

func RegisterDriver(resolverData map[string]*registry.ServiceInfo) *Driver {
	d := &Driver{
		cache: make(map[string]registry.Registry),
	}
	r := &Registry{
		data: resolverData,
	}
	d.cache["mock"] = r
	registry.RegisterDriver(scheme, d)
	return d

}

func (d *Driver) CreateRegistry(cnf *conf.Configuration) (registry.Registry, error) {
	return &Registry{
		data: make(map[string]*registry.ServiceInfo),
	}, nil
}

func (d *Driver) GetRegistry(name string) (registry.Registry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	r, ok := d.cache[name]
	if !ok {
		return nil, errors.New("registry not found,may not set it a reference registry")
	}
	return r, nil
}

func (d *Driver) ResolverBuilder(cnf *conf.Configuration) (resolver.Builder, error) {
	return &Resolver{
		data: d.cache["mock"].(*Registry).data,
	}, nil
}

func (d *Driver) WithDialOptions(registryOpts registry.DialOptions) ([]grpc.DialOption, error) {
	return nil, nil
}

type Registry struct {
	data map[string]*registry.ServiceInfo
	mu   sync.RWMutex
}

func (r *Registry) Register(info *registry.ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[info.Name] = info
	return nil
}

func (r *Registry) Unregister(info *registry.ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, info.Name)
	return nil
}

func (r *Registry) TTL() time.Duration {
	return 10 * time.Second
}

func (r *Registry) Close() {
}

func (r *Registry) GetServiceInfos(s string) ([]*registry.ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return []*registry.ServiceInfo{r.data[s]}, nil
}

type Resolver struct {
	data map[string]*registry.ServiceInfo
}

func (r *Resolver) ResolveNow(options resolver.ResolveNowOptions) {
	return
}

func (r *Resolver) Close() {
	return
}

func (r *Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	dp := target.URL.Host
	addr := r.data[dp].Address()
	err := cc.UpdateState(resolver.State{
		Addresses: []resolver.Address{
			{
				Addr: addr,
				Attributes: attributes.New(
					"is_mock", true,
				),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Resolver) Scheme() string {
	return scheme
}
