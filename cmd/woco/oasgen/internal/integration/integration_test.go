package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen"
	"os"
	"testing"
)

func TestGenerateServer(t *testing.T) {
	if os.Getenv("TEST_WIP") != "" {
		t.Skip("skipping test in short mode.")
	}
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
	if os.Getenv("TEST_WIP") != "" {
		t.Skip("skipping test in short mode.")
	}
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
