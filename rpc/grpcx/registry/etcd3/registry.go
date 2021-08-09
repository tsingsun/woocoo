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
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
	"strings"
	"sync"
	"time"
)

func init() {
	registry.RegisterDriver(scheme, func() registry.Registry {
		return New()
	})
}

type Config struct {
	EtcdConfig clientv3.Config `json:"etcd" yaml:"etcd"`
	TTL        time.Duration   `json:"ttl" yaml:"ttl"`
	logger     *zap.Logger
}

type Registry struct {
	sync.RWMutex
	config   Config
	client   *clientv3.Client
	register map[string]uint64
	leases   map[string]clientv3.LeaseID
}

func New() *Registry {
	return &Registry{
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
		config:   Config{},
	}
}

func (r *Registry) Apply(cfg *conf.Configuration, path string) {
	fixConfigDuration(cfg, path)
	if err := cfg.Parser().UnmarshalByJson(path, &r.config); err != nil {
		panic(err)
	}
	if cfg.IsSet("log") {
		r.config.logger = log.Operator()
	}
	if k := conf.Join(path, "etcd", "tls"); cfg.IsSet(k) {
		cp := cfg.String(conf.Join(k, "ssl_certificate"))
		kp := cfg.String(conf.Join(k, "ssl_certificate_key"))
		if cp != "" && kp != "" {
			r.config.EtcdConfig.TLS = registry.TLS(cfg.GetBaseDir(), cp, kp)
		} else {
			r.config.EtcdConfig.TLS = nil
		}
	} else {
		r.config.EtcdConfig.TLS = nil
	}
	err := r.buildClient()
	if err != nil {
		panic(err)
	}
}

func fixConfigDuration(cnf *conf.Configuration, path string) {
	for _, k := range []string{"ttl"} {
		if k := strings.Join([]string{path, k}, conf.KeyDelimiter); cnf.IsSet(k) {
			v := cnf.Duration(k)
			if v < time.Second {
				cnf.Parser().Set(k, v*time.Second)
			}
		}
	}
	//etcd config
	for _, k := range []string{"auto-sync-interval", "dial-timeout", "dial-keep-alive-time", "dial-keep-alive-timeout"} {
		if k := strings.Join([]string{path, "etcd", k}, conf.KeyDelimiter); cnf.IsSet(k) {
			v := cnf.Duration(k)
			if v < time.Second {
				cnf.Parser().Set(k, v*time.Second)
			}
		}
	}
}

func BuildFromConfig(config *Config) (*Registry, error) {
	r := &Registry{
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
	}
	if config == nil {
		r.config = Config{
			EtcdConfig: clientv3.Config{
				DialTimeout: 5 * time.Second,
			},
		}
	} else {
		r.config = *config
	}
	if err := r.buildClient(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) buildClient() (err error) {
	if r.config.logger != nil {
		r.config.EtcdConfig.Logger = r.config.logger
	}
	r.client, err = clientv3.New(r.config.EtcdConfig)
	return err
}

func nodePath(servicePath, id string) string {
	//name := strings.Replace(servicePath, "/", "-", -1)
	//v := strings.Replace(version, "/", "-", -1)
	//nid := strings.Replace(id, "/", "-", -1)
	return strings.Join([]string{servicePath, id}, "/")
}

func (r *Registry) Register(node *registry.NodeInfo) error {
	r.RLock()
	leaseID, ok := r.leases[node.ServiceLocation+node.ID]
	r.RUnlock()
	key := nodePath(node.ServiceLocation, node.ID)
	if !ok {
		ctx, cancle := context.WithTimeout(context.Background(), r.config.EtcdConfig.DialTimeout)
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
				var oldNode = new(registry.NodeInfo)
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
				r.leases[node.ServiceLocation+node.ID] = leaseID
				r.register[node.ServiceLocation+node.ID] = h
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
	oldh, ok := r.register[node.ServiceLocation+node.ID]
	r.Unlock()

	// the service is unchanged, skip registering
	if ok && oldh == h && !leaseNotFound {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.config.EtcdConfig.DialTimeout)
	defer cancel()

	var lgr *clientv3.LeaseGrantResponse
	if r.config.TTL.Seconds() > 0 {
		// get a lease used to expire keys since we have a ttl
		lgr, err = r.client.Grant(ctx, int64(r.config.TTL.Seconds()))
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
	r.register[node.ServiceLocation+node.ID] = h
	// save our leaseID of the service
	if lgr != nil {
		r.leases[node.ServiceLocation+node.ID] = lgr.ID
	}
	r.Unlock()

	return nil
}

// UnRegister remove service from etcd
func (r *Registry) Unregister(node *registry.NodeInfo) error {
	r.Lock()
	// delete our hash of the service
	delete(r.register, node.ServiceLocation+node.ID)
	// delete our lease of the service
	delete(r.leases, node.ServiceLocation+node.ID)
	r.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), r.config.EtcdConfig.DialTimeout)
	defer cancel()

	_, err := r.client.Delete(ctx, nodePath(node.ServiceLocation, node.ID))
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) Close() {
	r.client.Close()
}

func (r *Registry) TTL() time.Duration {
	return r.config.TTL
}

func (r *Registry) ResolverBuilder(serviceLocation string) resolver.Builder {
	return &etcdResolver{
		scheme:     scheme,
		etcdConfig: r.config.EtcdConfig,
		watchPath:  serviceLocation,
	}
}
