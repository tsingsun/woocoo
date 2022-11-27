package project

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func Test_generateWeb(t *testing.T) {
	type args struct {
		cfg *Config
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
					Package: "github.com/tsingsun/woocoo/example",
					Target: func() string {
						fd, err := filepath.Abs("/Users/tsingsun/Projects/wocotest")
						require.NoError(t, err)
						return fd
					}(),
					Modules: []string{"redis", "db", "otel", "web", "grpc"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := generateWeb(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("generateWeb() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
