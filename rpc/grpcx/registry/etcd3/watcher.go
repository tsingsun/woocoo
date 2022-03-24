package etcd3

import (
	"context"
	"encoding/json"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"sync"
)

type Watcher struct {
	key     string
	client  *clientv3.Client
	ctx     context.Context
	cancel  context.CancelFunc
	addrMap map[string]resolver.Address
	wg      sync.WaitGroup
}

func (w *Watcher) Close() {
	w.cancel()
}

func newWatcher(key string, cli *clientv3.Client) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watcher{
		key:     key,
		client:  cli,
		ctx:     ctx,
		cancel:  cancel,
		addrMap: map[string]resolver.Address{},
	}
	return w
}

func (w *Watcher) GetAllAddresses() (ret []resolver.Address) {
	resp, err := w.client.Get(w.ctx, w.key, clientv3.WithPrefix())
	if err != nil {
		logger.Error("GetAllAddresses error", zap.Error(err))
		return
	}
	nodeInfos := extractAddrs(resp)
	if len(nodeInfos) > 0 {
		for _, node := range nodeInfos {
			addr := resolver.Address{
				Addr:       node.Address,
				Attributes: node.ToAttributes(),
			}
			w.addrMap[node.BuildKey()] = addr
			//as := v.Pairs()
			ret = append(ret, addr)
		}
	}
	return
}

func (w *Watcher) Watch() chan []resolver.Address {
	out := make(chan []resolver.Address, 10)
	w.wg.Add(1)
	go func() {
		defer func() {
			close(out)
			w.wg.Done()
		}()
		w.GetAllAddresses()
		out <- w.cloneAddresses(w.getAddrs())

		rch := w.client.Watch(w.ctx, w.key, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				switch ev.Type {
				case mvccpb.PUT:
					node := registry.NodeInfo{}
					err := json.Unmarshal([]byte(ev.Kv.Value), &node)
					if err != nil {
						logger.Error("Parse node data error:", zap.Error(err))
						continue
					}
					if w.addNode(node) {
						out <- w.cloneAddresses(w.getAddrs())
					}
				case mvccpb.DELETE:
					node := registry.NodeInfo{}
					if ev.Kv.Value == nil {
						w.removeAddress(string(ev.Kv.Key))
					} else {
						err := json.Unmarshal([]byte(ev.Kv.Value), &node)
						if err != nil {
							logger.Error("Parse node data error:", zap.Error(err))
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

func extractAddrs(resp *clientv3.GetResponse) []registry.NodeInfo {
	var addrs []registry.NodeInfo

	if resp == nil || resp.Kvs == nil {
		return addrs
	}

	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			nodeData := registry.NodeInfo{}
			err := json.Unmarshal(v, &nodeData)
			if err != nil {
				grpclog.Info("Parse node data error:", err)
				continue
			}
			addrs = append(addrs, nodeData)
		}
	}
	return addrs
}

func (w *Watcher) cloneAddresses(in []resolver.Address) []resolver.Address {
	out := make([]resolver.Address, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = in[i]
	}
	return out
}

func (w *Watcher) addNode(node registry.NodeInfo) bool {
	addr := resolver.Address{Addr: node.Address, Attributes: node.ToAttributes()}
	w.addrMap[node.BuildKey()] = addr
	return true
}

func (w *Watcher) removeNode(node registry.NodeInfo) bool {
	delete(w.addrMap, node.BuildKey())
	return true
}

func (w *Watcher) removeAddress(key string) bool {
	delete(w.addrMap, key)
	return true
}

func (w *Watcher) getAddrs() (addrs []resolver.Address) {
	for _, address := range w.addrMap {
		addrs = append(addrs, address)
	}
	return
}
