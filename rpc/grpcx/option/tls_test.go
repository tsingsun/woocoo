package option

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

func TestTLSOption_Name(t *testing.T) {
	opt := TLSOption{}
	if opt.Name() != "tls" {
		t.Errorf("opt.Name() = %s, want tls", opt.Name())
	}
}

func TestTLSOption(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		opt := TLSOption{}
		cfg := conf.NewFromStringMap(map[string]any{})
		assert.Panics(t, func() {
			opt.ServerOption(cfg)
		})
	})
	t.Run("Error ServerOption files", func(t *testing.T) {
		opt := TLSOption{}
		cfg := conf.NewFromStringMap(map[string]any{
			"cert": "testdata/cert.pem",
			"key":  "testdata/key.pem",
		})
		assert.Panics(t, func() {
			opt.ServerOption(cfg)
		})
	})
	t.Run("ServerOption ok", func(t *testing.T) {
		opt := TLSOption{}
		cfg := conf.NewFromStringMap(map[string]any{
			"cert": "x509/client.crt",
			"key":  "x509/client.key",
		}, conf.WithBaseDir(testdata.BaseDir()))
		assert.NotNil(t, opt.ServerOption(cfg))
	})

	t.Run("DialOption ok", func(t *testing.T) {
		opt := TLSOption{}
		cfg := conf.NewFromStringMap(map[string]any{
			"cert": "x509/client.crt",
			"key":  "x509/client.key",
		}, conf.WithBaseDir(testdata.BaseDir()))
		assert.NotNil(t, opt.DialOption(cfg))
	})
	t.Run("insecure DialOption ok", func(t *testing.T) {
		opt := TLSOption{}
		cfg := conf.NewFromStringMap(map[string]any{
			"cert": "",
			"key":  "",
		})
		assert.NotNil(t, opt.DialOption(cfg))
	})
	t.Run("error cert", func(t *testing.T) {
		opt := TLSOption{}
		cfg := conf.NewFromStringMap(map[string]any{
			"cert": "x509/client.key",
			"key":  "",
		}, conf.WithBaseDir(testdata.BaseDir()))

		assert.Panics(t, func() {
			opt.DialOption(cfg)
		})
	})
}
