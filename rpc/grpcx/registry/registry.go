package registry

import (
	"crypto/tls"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"path/filepath"
	"time"
)

type registryNewFunc func() Registry

var (
	registryManager map[string]registryNewFunc = make(map[string]registryNewFunc)
)

func RegisterDriver(schema string, newFunc registryNewFunc) error {
	if _, ok := registryManager[schema]; ok {
		panic("registry storage type has exists:" + schema)
	}
	registryManager[schema] = newFunc
	return nil
}

func GetRegistry(schema string) Registry {
	if f, ok := registryManager[schema]; ok {
		return f()
	}
	panic("can not find registry:" + schema)
}

// The registry provides an interface for service discovery
type Registry interface {
	Register(nodeInfo *NodeInfo) error
	Unregister(nodeInfo *NodeInfo) error
	TTL() time.Duration
	Close()
	ResolverBuilder(serviceLocation string) resolver.Builder
}

type NodeInfo struct {
	ID              string `json:"id" yaml:"id"`
	ServiceLocation string
	ServiceVersion  string
	Address         string
	Metadata        metadata.MD
}

func (n NodeInfo) Pairs() []interface{} {
	var val []interface{}
	if n.Metadata.Len() == 0 {
		return val
	}
	for _, strings := range n.Metadata {
		for _, s := range strings {
			val = append(val, s)
		}
	}
	return val
}

func TLS(basedir, ssl_certificate, ssl_certificate_key string) *tls.Config {
	if !filepath.IsAbs(ssl_certificate) {
		ssl_certificate = filepath.Join(basedir, ssl_certificate)
	}
	if !filepath.IsAbs(ssl_certificate_key) {
		ssl_certificate_key = filepath.Join(basedir, ssl_certificate_key)
	}
	cer, err := tls.LoadX509KeyPair(ssl_certificate, ssl_certificate_key)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cer}}
}
