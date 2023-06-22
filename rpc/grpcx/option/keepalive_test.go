package option

import (
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
