package polarismesh

import (
	"fmt"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/tsingsun/woocoo/pkg/conf"
	"strings"
	"sync"
)

var (
	// DefaultNamespace default namespace when namespace is not set
	DefaultNamespace = "default"
	// LoadBalanceConfig config for do the balance
	LoadBalanceConfig = fmt.Sprintf("{\n  \"loadBalancingConfig\": [ { \"%s\": {} } ]}", scheme)
)

var (
	polarisCallerServiceKey   = "polaris.request.caller.service"
	polarisCallerNamespaceKey = "polaris.request.caller.namespace"
	registerServiceTokenKey   = "token"
)

var (
	polarisContext      api.SDKContext
	polarisConfig       config.Configuration
	mutexPolarisContext sync.Mutex
	oncePolarisConfig   sync.Once
)

// PolarisContext get or init the global polaris context
func PolarisContext() (ctx api.SDKContext, err error) {
	mutexPolarisContext.Lock()
	defer mutexPolarisContext.Unlock()
	if polarisContext != nil {
		return polarisContext, nil
	}
	polarisContext, err = api.InitContextByConfig(PolarisConfig())
	return polarisContext, err
}

// PolarisConfig get or init the global polaris configuration
func PolarisConfig() config.Configuration {
	if polarisConfig == nil {
		oncePolarisConfig.Do(func() {
			polarisConfig = api.NewConfiguration()
		})
	}
	return polarisConfig
}

// NewPolarisConfig create a polaris configuration from conf.Configuration
func NewPolarisConfig(cfg *conf.Configuration) (config.Configuration, error) {
	var (
		parser *conf.Parser
		err    error
	)
	pcfg := cfg.Sub("polaris")
	if pcfg.IsSet("configFile") {
		parser, err = conf.NewParserFromFile(cfg.Abs(pcfg.String("configFile")))
		if err != nil {
			return nil, err
		}
	} else {
		parser = pcfg.Parser()
	}
	bts, err := parser.ToBytes(yaml.Parser())
	if err != nil {
		return nil, err
	}
	pc, err := config.LoadConfiguration(bts)
	if err != nil {
		return nil, err
	}
	return pc, nil
}

// SetPolarisConfig set the global polaris configuration
func SetPolarisConfig(cfg *conf.Configuration) (err error) {
	polarisConfig, err = NewPolarisConfig(cfg)
	if err != nil {
		return err
	}
	return nil
}

func extractBareMethodName(fullMethodName string) string {
	index := strings.LastIndex(fullMethodName, "/")
	if index == -1 {
		return ""
	}
	return fullMethodName[index+1:]
}

func extractBareServiceName(fullMethodName string) string {
	index := strings.LastIndex(fullMethodName, "/")
	if index == -1 {
		return ""
	}
	return fullMethodName[:index]
}
