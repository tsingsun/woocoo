package option

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// KeepAliveOption keepalive option
type KeepAliveOption struct {
}

func (KeepAliveOption) Name() string {
	return "keepalive"
}

func (KeepAliveOption) ServerOption(cfg *conf.Configuration) grpc.ServerOption {
	sp := keepalive.ServerParameters{}
	if err := cfg.Unmarshal(&sp); err != nil {
		panic(err)
	}
	return grpc.KeepaliveParams(sp)
}

func (KeepAliveOption) DialOption(cfg *conf.Configuration) grpc.DialOption {
	sp := keepalive.ClientParameters{}
	if err := cfg.Unmarshal(&sp); err != nil {
		panic(err)
	}
	return grpc.WithKeepaliveParams(sp)
}

// KeepaliveEnforcementPolicy keepalive enforcement policy for the server.
func KeepaliveEnforcementPolicy(cfg *conf.Configuration) grpc.ServerOption {
	ep := keepalive.EnforcementPolicy{}
	if err := cfg.Unmarshal(&ep); err != nil {
		panic(err)
	}

	return grpc.KeepaliveEnforcementPolicy(ep)
}
