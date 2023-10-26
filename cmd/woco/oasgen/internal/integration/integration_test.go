package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen"
	"testing"
)

func TestGenerateServer(t *testing.T) {
	cfgPath := "config.yaml"
	path := "petstore.yaml"
	cfg := &oasgen.Config{
		OpenAPISchema: path,
		Package:       "petstore",
		Target:        "petstore-server",
	}
	err := oasgen.LoadConfig(cfg, cfgPath)
	assert.NoError(t, err)
	err = oasgen.Generate(path, cfg)
	assert.NoError(t, err)
}

func TestGenerateClient(t *testing.T) {
	cfgPath := "config.yaml"
	path := "petstore.yaml"
	cfg := &oasgen.Config{
		OpenAPISchema: path,
		Package:       "client",
		Target:        "petstore-client",
		GenClient:     true,
	}
	err := oasgen.LoadConfig(cfg, cfgPath)
	assert.NoError(t, err)
	err = oasgen.Generate(path, cfg)
	assert.NoError(t, err)
}
