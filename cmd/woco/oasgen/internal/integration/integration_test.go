package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen"
	"testing"
)

func TestGenerate(t *testing.T) {
	cfgPath := "config.yaml"
	path := "petstore.yaml"
	cfg := &oasgen.Config{
		OpenAPISchema: path,
		Target:        "petstore",
	}
	err := oasgen.LoadConfig(cfg, cfgPath)
	assert.NoError(t, err)
	err = oasgen.Generate(path, cfg)
	assert.NoError(t, err)
}
