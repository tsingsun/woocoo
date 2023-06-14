package polarismesh

import (
	"errors"
	"fmt"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/tsingsun/woocoo/pkg/conf"
	"strings"
	"sync"
)

const (
	polarisRequestLbHashKey = "polaris.balancer.request.hashKey"
	polarisRequestLbPolicy  = "polaris.balancer.request.lbPolicy"
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
	mutexPolarisContext sync.Mutex
	oncePolarisContext  sync.Once
)

// PolarisContext get or init the global polaris context
func PolarisContext() (ctx api.SDKContext, err error) {
	mutexPolarisContext.Lock()
	defer mutexPolarisContext.Unlock()
	if polarisContext != nil {
		return polarisContext, nil
	}
	return nil, errors.New("PolarisContext:polaris context not init")
}

// SetPolarisContextOnceByConfig set polaris context by config,if polaris context has init then do nothing
func SetPolarisContextOnceByConfig(cfg config.Configuration) (err error) {
	mutexPolarisContext.Lock()
	defer mutexPolarisContext.Unlock()
	if polarisContext == nil {
		polarisContext, err = api.InitContextByConfig(cfg)
		if err != nil {
			return err
		}
	}
	return nil
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
	if cfg.Bool("global") {
		if err = SetPolarisContextOnceByConfig(pc); err != nil {
			return nil, err
		}
	}
	return pc, nil
}

func extractBareMethodName(fullMethodName string) string {
	index := strings.LastIndex(fullMethodName, "/")
	if index == -1 {
		return ""
	}
	return fullMethodName[index+1:]
}
