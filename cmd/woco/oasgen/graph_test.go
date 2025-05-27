package oasgen

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestNewGraph(t *testing.T) {
	t.Run("persotre", func(t *testing.T) {
		c := &Config{
			OpenAPISchema: "./internal/integration/petstore.yaml",
		}
		fs, err := filepath.Abs("./internal/integration/petstore.yaml")
		require.NoError(t, err)
		s, err := openapi3.NewLoader().LoadFromFile(fs)
		require.NoError(t, err)
		g, err := NewGraph(c, s)
		require.NoError(t, err)
		assert.Equal(t, 19, len(g.Schemas))
		assert.Equal(t, 3, len(g.Nodes))
	})
}
