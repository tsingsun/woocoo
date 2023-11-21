package etcd3

import (
	"context"
	"encoding/json"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"strings"
)

const scheme = "etcd"

type etcdBuilder struct {
	etcdConfig clientv3.Config
}

// Build Implement the Build method in the Resolver Builder interface
func (r *etcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	etcdCli, err := clientv3.New(r.etcdConfig)
	if err != nil {
		return nil, err
	}
	options, err := registry.TargetToOptions(target) // nolint: staticcheck
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	wt := &etcdResolver{
		client:  etcdCli,
		ctx:     ctx,
		cancel:  cancel,
		cc:      cc,
		target:  target,
		options: options,
		rsCh:    make(chan struct{}, 1),
		addrMap: map[string]resolver.Address{},
	}
	go wt.Watch()
	wt.ResolveNow(resolver.ResolveNowOptions{})
	return wt, nil
}

func (r *etcdBuilder) Scheme() string {
	return scheme
}

// RegisterResolver register etcdBuilder as the global grpc resolver
func RegisterResolver(etcdConfig clientv3.Config) {
	resolver.Register(&etcdBuilder{
		etcdConfig: etcdConfig,
	})
}

type etcdResolver struct {
	key     string
	client  *clientv3.Client
	ctx     context.Context
	cancel  context.CancelFunc
	cc      resolver.ClientConn
	target  resolver.Target
	options *registry.DialOptions
	addrMap map[string]resolver.Address
	rsCh    chan struct{}
}

func (w *etcdResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	select {
	case w.rsCh <- struct{}{}:
	default:
	}
}

func (w *etcdResolver) Close() {
	w.cancel()
}

// safeKey make sure the key is start and end with '/'. keep the watch key is exactly path in etcd.
func (w *etcdResolver) safeKey() {
	if w.options.Namespace != "" {
		w.key = strings.Join([]string{w.options.Namespace, w.options.ServiceName}, "/")
	} else {
		w.key = w.options.ServiceName
	}
	if !strings.HasPrefix(w.key, "/") {
		w.key = "/" + w.key
	}
	if !strings.HasSuffix(w.key, "/") {
		w.key += "/"
	}
}

func (w *etcdResolver) Watch() {
	w.safeKey()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.rsCh:
		}
		out := w.watchChan()
		for addrs := range out {
			if len(addrs) == 0 {
				grpclog.Errorf("resolver got zero addresses:%s", w.key)
			}

			if err := w.cc.UpdateState(resolver.State{Addresses: addrs}); err != nil {
				grpclog.Errorf("fail to do update service %s: %v", w.target.URL.String(), err)
			}
		}
	}
}

func (w *etcdResolver) GetAllAddresses() (ret []resolver.Address) {
	resp, err := w.client.Get(w.ctx, w.key, clientv3.WithPrefix())
	if err != nil {
		grpclog.Errorf("GetAllAddresses error:%v", err)
		return
	}
	nodeInfos, _ := extractAddrs(resp)
	if len(nodeInfos) > 0 {
		for _, node := range nodeInfos {
			addr := resolver.Address{
				Addr:       node.Address(),
				Attributes: node.ToAttributes(),
			}
			w.addrMap[node.BuildKey()] = addr
			ret = append(ret, addr)
		}
	}
	return
}

func (w *etcdResolver) watchChan() chan []resolver.Address {
	out := make(chan []resolver.Address, 10)
	go func() {
		defer func() {
			close(out)
		}()
		w.GetAllAddresses()
		out <- w.cloneAddresses(w.getAddrs())

		rch := w.client.Watch(w.ctx, w.key, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				switch ev.Type {
				case mvccpb.PUT:
					node := registry.ServiceInfo{}
					err := json.Unmarshal(ev.Kv.Value, &node)
					if err != nil {
						grpclog.Errorf("Parse node data error:%v", err)
						continue
					}
					if w.addNode(node) {
						out <- w.cloneAddresses(w.getAddrs())
					}
				case mvccpb.DELETE:
					node := registry.ServiceInfo{}
					if ev.Kv.Value == nil {
						w.removeAddress(string(ev.Kv.Key))
					} else {
						err := json.Unmarshal(ev.Kv.Value, &node)
						if err != nil {
							grpclog.Errorf("Parse node data error:%v", err)
							continue
						}
						if w.removeNode(node) {
							out <- w.cloneAddresses(w.getAddrs())
						}
					}
					out <- w.cloneAddresses(w.getAddrs())
				}
			}
		}
	}()
	return out
}

func extractAddrs(resp *clientv3.GetResponse) (infos []*registry.ServiceInfo, err error) {
	if resp == nil || resp.Kvs == nil {
		return
	}
	var lasterr error
	infos = make([]*registry.ServiceInfo, 0, len(resp.Kvs))
	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			nodeData := &registry.ServiceInfo{}
			err = json.Unmarshal(v, &nodeData)
			if err != nil {
				lasterr = err
				grpclog.Info("Parse node data error:", err)
				continue
			}
			infos = append(infos, nodeData)
		}
	}
	err = lasterr
	return
}

func (w *etcdResolver) cloneAddresses(in []resolver.Address) []resolver.Address {
	out := make([]resolver.Address, len(in))
	copy(out, in)
	return out
}

func (w *etcdResolver) addNode(node registry.ServiceInfo) bool {
	addr := resolver.Address{Addr: node.Address(), Attributes: node.ToAttributes()}
	w.addrMap[node.BuildKey()] = addr
	return true
}

func (w *etcdResolver) removeNode(node registry.ServiceInfo) bool {
	delete(w.addrMap, node.BuildKey())
	return true
}

func (w *etcdResolver) removeAddress(key string) bool {
	delete(w.addrMap, key)
	return true
}

func (w *etcdResolver) getAddrs() (addrs []resolver.Address) {
	for _, address := range w.addrMap {
		addrs = append(addrs, address)
	}
	return
}
