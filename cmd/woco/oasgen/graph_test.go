package oasgen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestNewGraph(t *testing.T) {
	type args struct {
		c      *Config
		schema *openapi3.T
	}
	tests := []struct {
		name    string
		args    args
		wantG   *Graph
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "petstore",
			args: args{
				c: &Config{
					OpenAPISchema: "./internal/integration/petstore.yaml",
				},
				schema: func() *openapi3.T {
					fs, err := filepath.Abs("./internal/integration/petstore.yaml")
					require.NoError(t, err)
					s, err := openapi3.NewLoader().LoadFromFile(fs)
					require.NoError(t, err)
					return s
				}(),
			},
			wantG:   nil,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotG, err := NewGraph(tt.args.c, tt.args.schema)
			if !tt.wantErr(t, err, fmt.Sprintf("NewGraph(%v, %v)", tt.args.c, tt.args.schema)) {
				return
			}
			assert.Equalf(t, tt.wantG, gotG, "NewGraph(%v, %v)", tt.args.c, tt.args.schema)
		})
	}
}
