package project

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func Test_Generate(t *testing.T) {
	type args struct {
		cfg  *Config
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				cfg: &Config{
					Package: "github.com/tsingsun/woocoo/alltest",
					Header:  "//go:build ignore",
					Target: func() string {
						fd, err := filepath.Abs("internal/integration/alltest")
						require.NoError(t, err)
						return fd
					}(),
					Modules: []string{"otel", "web", "grpc"},
				},
			},
		},
		{
			name: "empty-module",
			args: args{
				cfg: &Config{
					Package: "github.com/tsingsun/wocotest",
					Header:  "//go:build ignore",
					Target: func() string {
						fd, err := filepath.Abs("internal/integration/wocotest")
						require.NoError(t, err)
						return fd
					}(),
					Modules: []string{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Generate(tt.args.cfg, tt.args.opts...); (err != nil) != tt.wantErr {
				t.Errorf("generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
