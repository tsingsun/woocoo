package option

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"testing"
)

func TestCompressionOption_ServerOption(t *testing.T) {
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name  string
		args  args
		panic bool
		check func(grpc.ServerOption)
	}{
		{
			name: "gzip",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"name":  "gzip",
					"level": 1,
				}),
			},
			check: func(opt grpc.ServerOption) {
				assert.Nil(t, opt)
			},
		},
		{
			name: "no exist",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"name":  "none",
					"level": 1,
				}),
			},
			panic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			co := CompressionOption{}
			assert.Equal(t, "compression", co.Name())
			if tt.panic {
				assert.Panics(t, func() {
					co.ServerOption(tt.args.cfg)
				})
			} else {
				tt.check(co.ServerOption(tt.args.cfg))
			}
		})
	}
}

func TestCompressionOption_DialOption(t *testing.T) {
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "gzip",
			args: args{
				cfg: conf.NewFromStringMap(map[string]any{
					"name":  "gzip",
					"level": 1,
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			co := CompressionOption{}
			got := co.DialOption(tt.args.cfg)
			assert.NotNil(t, got)
		})
	}
}
