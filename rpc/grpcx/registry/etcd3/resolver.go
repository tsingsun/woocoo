package etcd3

import (
	"fmt"
	"github.com/tsingsun/woocoo/pkg/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
	"strings"
	"sync"
)

const scheme = "etcd"

var (
	logger = log.Component("etcdRegistry")
)

type etcdResolver struct {
	scheme     string
	etcdConfig clientv3.Config
	watchPath  string
	watcher    *Watcher
	cc         resolver.ClientConn
	wg         sync.WaitGroup
}

func servicePath(groupName, serviceName string) string {
	g := strings.Replace(groupName, "/", "-", -1)
	s := strings.Replace(serviceName, "/", "-", -1)
	return "/" + strings.Join([]string{g, s}, "/")
}

func (r *etcdResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	etcdCli, err := clientv3.New(r.etcdConfig)
	if err != nil {
		return nil, err
	}
	r.cc = cc
	r.watcher = newWatcher(r.watchPath, etcdCli)
	r.start()
	return r, nil
}

func (r *etcdResolver) Scheme() string {
	return r.scheme
}

func (r *etcdResolver) start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		out := r.watcher.Watch()
		for addr := range out {
			r.cc.UpdateState(resolver.State{Addresses: addr})
			if len(addr) == 0 {
				logger.Error(fmt.Sprintf("resolver got zero addresses:%s", r.watchPath))
			}
		}
	}()
}

func (r *etcdResolver) ResolveNow(o resolver.ResolveNowOptions) {
}

func (r *etcdResolver) Close() {
	r.watcher.Close()
	r.wg.Wait()
}

func RegisterResolver(etcdConfig clientv3.Config, serviceLocation string) {
	resolver.Register(&etcdResolver{
		scheme:     scheme,
		etcdConfig: etcdConfig,
		watchPath:  serviceLocation,
	})
}
