package etcd3

import (
	"context"
	"encoding/json"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

func init() {
	registry.RegisterDriver(scheme, &Driver{cache: make(map[string]registry.Registry)})
}

// Driver is an etcd3 registry driver. Support reference config.
type Driver struct {
	cache map[string]registry.Registry
	mu    sync.RWMutex
}

// CreateRegistry creates a new registry.
func (drv *Driver) CreateRegistry(cfg *conf.Configuration) (registry.Registry, error) {
	drv.mu.Lock()
	defer drv.mu.Unlock()
	ccfg := cfg
	ref := cfg.String("ref")
	if ref != "" {
		if v, ok := drv.cache[ref]; ok {
			return v, nil
		}
		ccfg = cfg.Root().Sub(ref)
	}
	r := New()
	r.Apply(ccfg)
	if ref != "" {
		drv.cache[ref] = r
	}
	return r, nil
}

// ResolverBuilder creates a new resolver builder.
func (drv *Driver) ResolverBuilder(cfg *conf.Configuration) (resolver.Builder, error) {
	drv.mu.Lock()
	defer drv.mu.Unlock()
	var opts Options
	ccfg := cfg
	if ref := cfg.String("ref"); ref != "" {
		ccfg = cfg.Root().Sub(ref)
	}
	if err := ccfg.Unmarshal(&opts); err != nil {
		return nil, err
	}
	return &etcdBuilder{
		etcdConfig: opts.EtcdConfig,
	}, nil
}

// WithDialOptions no need to implement
func (drv *Driver) WithDialOptions(registry.DialOptions) ([]grpc.DialOption, error) {
	return nil, nil
}

type Options struct {
	EtcdConfig clientv3.Config `json:"etcd" yaml:"etcd"`
	TTL        time.Duration   `json:"ttl" yaml:"ttl"`
}

// Registry is an etcd3 registry for service discovery.
type Registry struct {
	sync.RWMutex
	opts     Options
	client   *clientv3.Client
	register map[string]uint64
	leases   map[string]clientv3.LeaseID
}

func New() *Registry {
	return &Registry{
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
		opts:     Options{},
	}
}

func (r *Registry) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(&r.opts); err != nil {
		panic(err)
	}
	if k := "etcd.tls"; cfg.IsSet(k) {
		tls, err := conf.NewTLS(cfg.Sub(k)).BuildTlsConfig()
		if err != nil {
			panic(err)
		}
		r.opts.EtcdConfig.TLS = tls
	}
	err := r.buildClient()
	if err != nil {
		panic(err)
	}
}

func BuildFromConfig(config *Options) (*Registry, error) {
	r := &Registry{
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
	}
	if config == nil {
		r.opts = Options{
			EtcdConfig: clientv3.Config{
				DialTimeout: 5 * time.Second,
			},
		}
	} else {
		r.opts = *config
	}
	if err := r.buildClient(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) buildClient() (err error) {
	r.opts.EtcdConfig.Logger = log.Global().Logger().Operator()
	r.client, err = clientv3.New(r.opts.EtcdConfig)
	return err
}

func (r *Registry) Register(node *registry.ServiceInfo) error {
	key := node.BuildKey()
	r.RLock()
	leaseID, ok := r.leases[key]
	r.RUnlock()
	if !ok {
		ctx, cancle := context.WithTimeout(context.Background(), r.opts.EtcdConfig.DialTimeout)
		defer cancle()
		// look for the existing key
		rsp, err := r.client.Get(ctx, key, clientv3.WithSerializable())
		if err != nil {
			return err
		}
		// get the existing lease
		for _, kv := range rsp.Kvs {
			if kv.Lease > 0 {
				leaseID = clientv3.LeaseID(kv.Lease)

				// decode the existing node,if the value unmarshal error,will ignore
				var oldNode *registry.ServiceInfo
				_ = json.Unmarshal(kv.Value, &oldNode)
				if oldNode == nil {
					continue
				}

				// create hash of service; uint64
				h, err := hashstructure.Hash(oldNode, hashstructure.FormatV2, nil)
				if err != nil {
					continue
				}

				// save the info
				r.Lock()
				r.leases[node.BuildKey()] = leaseID
				r.register[node.BuildKey()] = h
				r.Unlock()

				break
			}
		}
	}

	var leaseNotFound bool
	if leaseID > 0 {
		if _, err := r.client.KeepAliveOnce(context.TODO(), leaseID); err != nil {
			if err != rpctypes.ErrLeaseNotFound {
				return err
			}

			// lease not found do register
			leaseNotFound = true
		}
	}
	h, err := hashstructure.Hash(node, hashstructure.FormatV2, nil)
	if err != nil {
		return err
	}
	// get existing hash for the service node
	r.Lock()
	oldh, ok := r.register[node.BuildKey()]
	r.Unlock()

	// the service is unchanged, skip registering
	if ok && oldh == h && !leaseNotFound {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.opts.EtcdConfig.DialTimeout)
	defer cancel()

	var lgr *clientv3.LeaseGrantResponse
	if r.opts.TTL.Seconds() > 0 {
		// get a lease used to expire keys since we have a ttl
		lgr, err = r.client.Grant(ctx, int64(r.opts.TTL.Seconds()))
		if err != nil {
			return err
		}
	}

	// create an entry for the node
	val, err := json.Marshal(node)
	if err != nil {
		return err
	}
	value := string(val)
	if lgr != nil {
		_, err = r.client.Put(ctx, key, value, clientv3.WithLease(lgr.ID))
	} else {
		_, err = r.client.Put(ctx, key, value)
	}
	if err != nil {
		return err
	}

	r.Lock()
	// save our hash of the service
	r.register[node.BuildKey()] = h
	// save our leaseID of the service
	if lgr != nil {
		r.leases[node.BuildKey()] = lgr.ID
	}
	r.Unlock()

	return nil
}

// Unregister remove service from etcd
func (r *Registry) Unregister(node *registry.ServiceInfo) error {
	r.Lock()
	// delete our hash of the service
	delete(r.register, node.BuildKey())
	// delete our lease of the service
	delete(r.leases, node.BuildKey())
	r.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), r.opts.EtcdConfig.DialTimeout)
	defer cancel()

	_, err := r.client.Delete(ctx, node.BuildKey())
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) Close() {
	r.client.Close()
}

func (r *Registry) TTL() time.Duration {
	return r.opts.TTL
}
