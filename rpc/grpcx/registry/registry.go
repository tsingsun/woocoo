package registry

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const OptionKey = "options"

var (
	registryManager = make(map[string]Registry)
)

// RegisterDriver register a new Registry could be: conf.Configurable,and will be lazy loaded in server.Apply function
func RegisterDriver(scheme string, reg Registry) {
	registryManager[scheme] = reg
}

// DialOption is the options for client dial when using registry resolver.
type DialOption interface {
	apply(options *DialOptions)
}

// DialOptions is the options for client dial when using registry resolver.
type DialOptions struct {
	GRPCDialOptions []grpc.DialOption `json:"-" yaml:"-"`
	Namespace       string            `json:"namespace" yaml:"namespace"`
	ServiceName     string            `json:"serviceName" yaml:"serviceName"`
	Metadata        map[string]string `json:"metadata" yaml:"metadata"`
}

func TargetToOptions(target resolver.Target) (*DialOptions, error) {
	options := &DialOptions{}
	if len(target.URL.RawQuery) > 0 {
		var optionsStr string
		values := target.URL.Query()
		if len(values) > 0 {
			optionValues := values[OptionKey]
			if len(optionValues) > 0 {
				optionsStr = optionValues[0]
			}
		}
		if len(optionsStr) > 0 {
			value, err := base64.URLEncoding.DecodeString(optionsStr)
			if nil != err {
				return nil, fmt.Errorf(
					"fail to decode endpoint %s, options %s: %v", target.URL.Path, optionsStr, err)
			}
			if err = json.Unmarshal(value, options); nil != err {
				return nil, fmt.Errorf("fail to unmarshal options %s: %v", string(value), err)
			}
		}
	} else {
		options.Namespace = target.URL.Host
		options.ServiceName = target.Endpoint
	}
	return options, nil
}

func GetRegistry(scheme string) Registry {
	if f, ok := registryManager[scheme]; ok {
		return f
	}
	panic("can not find registry:" + scheme)
}

// Registry provides an interface for service discovery
type Registry interface {
	// Register a service node
	Register(serviceInfo *ServiceInfo) error
	// Unregister a service node
	Unregister(serviceInfo *ServiceInfo) error
	// TTL returns the time to live of the service node, if it is not available, return 0.
	// every tick will call Register function to refresh.
	TTL() time.Duration
	Close()
	// ResolverBuilder returns a resolver.Builder.
	ResolverBuilder(config *conf.Configuration) resolver.Builder
}

// ServiceInfo is the service information
type ServiceInfo struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Version   string            `json:"version" yaml:"version"`
	Host      string            `json:"host" yaml:"host"`
	Port      int               `json:"port" yaml:"port"`
	Protocol  string            `json:"protocol" yaml:"protocol"`
	Metadata  map[string]string `json:"metadata" yaml:"metadata"`
}

func (si ServiceInfo) ToAttributes() *attributes.Attributes {
	var val *attributes.Attributes
	for k, v := range si.Metadata {
		val.WithValue(k, v)
	}
	return val
}

// Address is the address of the service,example: host:port,ip:port
func (si ServiceInfo) Address() string {
	return si.Host + ":" + strconv.Itoa(si.Port)
}

func (si ServiceInfo) BuildKey() string {
	return nodePath(si.Namespace, si.Name, si.Version, si.Address())
}

// return service instance key
func nodePath(namespace, name, version, addr string) string {
	return strings.Join([]string{namespace, name, version, addr}, "/")
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
