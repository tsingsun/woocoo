package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/codegen"
	"testing"
)

func TestGenerate(t *testing.T) {
	cfgPath := "config.yaml"
	path := "petstore.yaml"
	cfg := &codegen.Config{
		OpenAPISchema: path,
		Package:       "petstore",
		//PkgPath:       "github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore",
	}
	err := oasgen.LoadConfig(cfg, cfgPath)
	cfg.Target = "petstore"
	assert.NoError(t, err)
	err = oasgen.Generate(path, cfg)
	assert.NoError(t, err)
}
