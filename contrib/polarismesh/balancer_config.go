package polarismesh

import "google.golang.org/grpc/serviceconfig"

// LBConfig is the LB config for the polaris policy.
type LBConfig struct {
	serviceconfig.LoadBalancingConfig `json:"-"`
	HashKey                           string `json:"hash_key"`
	LbPolicy                          string `json:"lb_policy"`
}
