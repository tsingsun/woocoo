package option

import (
	"google.golang.org/grpc"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
)

func TestKeepAliveOption_Name(t *testing.T) {
	opt := KeepAliveOption{}
	if opt.Name() != "keepalive" {
		t.Errorf("opt.Name() = %s, want keepalive", opt.Name())
	}
}

func TestKeepAliveOption_ServerOption(t *testing.T) {
	cfg := conf.NewFromStringMap(map[string]any{
		"Time":                10 * time.Second,
		"Timeout":             5 * time.Second,
		"PermitWithoutStream": true,
	})
	opt := KeepAliveOption{}
	serverOpt := opt.ServerOption(cfg)
	assert.NotNil(t, serverOpt)
}

func TestKeepAliveOption_DialOption(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		cfg := conf.NewFromStringMap(map[string]any{})
		opt := KeepAliveOption{}
		dialOpt := opt.DialOption(cfg)
		assert.NotNil(t, dialOpt)
	})
	t.Run("not empty", func(t *testing.T) {
		cfg := conf.NewFromStringMap(map[string]any{
			"Time":                10 * time.Second,
			"Timeout":             5 * time.Second,
			"PermitWithoutStream": true,
		})
		opt := KeepAliveOption{}
		dialOpt := opt.DialOption(cfg)
		assert.NotNil(t, dialOpt)
	})
}

func TestKeepaliveEnforcementPolicy(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		cfg := conf.NewFromStringMap(map[string]any{
			"minTime":             "5s",
			"permitWithoutStream": "true",
		})
		opt := KeepaliveEnforcementPolicy(cfg)
		srv := grpc.NewServer(opt)
		assert.NotNil(t, srv)
	})
	t.Run("error time", func(t *testing.T) {
		cfg := conf.NewFromStringMap(map[string]any{
			"minTime":             "X",
			"permitWithoutStream": "true",
		})
		assert.Panics(t, func() {
			KeepaliveEnforcementPolicy(cfg)
		})
	})
}
