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
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

func init() {
	registry.RegisterDriver(scheme, New())
}

type Options struct {
	EtcdConfig clientv3.Config `json:"etcd" yaml:"etcd"`
	TTL        time.Duration   `json:"ttl" yaml:"ttl"`
}

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
	if k := conf.Join("etcd", "tls"); cfg.IsSet(k) {
		cp := cfg.String(conf.Join(k, "ssl_certificate"))
		kp := cfg.String(conf.Join(k, "ssl_certificate_key"))
		if cp != "" && kp != "" {
			r.opts.EtcdConfig.TLS = registry.TLS(cfg.Root().GetBaseDir(), cp, kp)
		} else {
			r.opts.EtcdConfig.TLS = nil
		}
	} else {
		r.opts.EtcdConfig.TLS = nil
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
	r.opts.EtcdConfig.Logger = log.Global().Operator()
	r.client, err = clientv3.New(r.opts.EtcdConfig)
	return err
}

func (r *Registry) Register(node *registry.ServiceInfo) error {
	r.RLock()
	leaseID, ok := r.leases[node.BuildKey()]
	r.RUnlock()
	key := node.BuildKey()
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

				// decode the existing node
				var oldNode = new(registry.ServiceInfo)
				err := json.Unmarshal(kv.Value, oldNode)
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

func (r *Registry) ResolverBuilder(config *conf.Configuration) resolver.Builder {
	return &etcdBuilder{
		etcdConfig: r.opts.EtcdConfig,
	}
}
