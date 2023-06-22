package polarismesh

import (
	"errors"
	"fmt"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/tsingsun/woocoo/pkg/conf"
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

// InitPolarisContext create polaris context by config
func InitPolarisContext(cnf *conf.Configuration) (cfg config.Configuration, ctx api.SDKContext, err error) {
	var (
		parser *conf.Parser
	)
	pcnf := cnf.Sub("polaris")
	if pcnf.IsSet("configFile") {
		parser, err = conf.NewParserFromFile(cnf.Abs(pcnf.String("configFile")))
		if err != nil {
			return nil, nil, err
		}
	} else {
		parser = pcnf.Parser()
	}
	bts, err := parser.ToBytes(yaml.Parser())
	if err != nil {
		return
	}
	cfg, err = config.LoadConfiguration(bts)
	if err != nil {
		return
	}
	// default global
	if cnf.IsSet("global") && !cnf.Bool("global") {
		ctx, err = api.InitContextByConfig(cfg)
	} else {
		if err = SetPolarisContextOnceByConfig(cfg); err != nil {
			return nil, nil, err
		}
		ctx, err = PolarisContext()
	}
	return
}
