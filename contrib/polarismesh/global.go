package polarismesh

import (
	"fmt"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/tsingsun/woocoo/pkg/conf"
	"sync"
)

var (
	// DefaultNamespace default namespace when namespace is not set
	DefaultNamespace = "default"
	// DefaultTTL default ttl value when ttl is not set
	DefaultTTL = 20
	// LoadBalanceConfig config for do the balance
	LoadBalanceConfig = fmt.Sprintf("{\n  \"loadBalancingConfig\": [ { \"%s\": {} } ]}", scheme)
)

var (
	polarisContext      api.SDKContext
	polarisConfig       config.Configuration
	mutexPolarisContext sync.Mutex
	oncePolarisConfig   sync.Once
)

// PolarisContext get or init the global polaris context
func PolarisContext(cfg *conf.Configuration) (ctx api.SDKContext, err error) {
	mutexPolarisContext.Lock()
	defer mutexPolarisContext.Unlock()
	if nil != polarisContext {
		return polarisContext, nil
	}
	polarisContext, err = api.InitContextByConfig(PolarisConfig(cfg))
	return polarisContext, err
}

// PolarisConfig get or init the global polaris configuration
func PolarisConfig(cfg *conf.Configuration) config.Configuration {
	oncePolarisConfig.Do(func() {
		bts, err := cfg.Sub("polaris").Parser().ToBytes(yaml.Parser())
		if nil != err {
			panic(err)
		}
		polarisConfig, err = config.LoadConfiguration(bts)
		if nil != err {
			panic(err)
		}
	})
	return polarisConfig
}
