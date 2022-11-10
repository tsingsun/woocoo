package oasgen

import (
	"context"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/codegen"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go/token"
	"golang.org/x/tools/go/packages"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func loadSwagger(filePath string) (oas *openapi3.T, err error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	u, err := url.Parse(filePath)
	if err != nil {
		panic(err)
	}
	if u.Scheme != "" && u.Host != "" {
		oas, err = loader.LoadFromURI(u)
	} else {
		oas, err = loader.LoadFromFile(filePath)
	}
	err = oas.Validate(context.Background())
	return
}

// LoadConfig reads the openapi3 config file
func LoadConfig(cfg *codegen.Config, filename string) (err error) {
	cnf, err := conf.NewParserFromFile(filename)
	if err != nil {
		return err
	}
	err = cnf.Unmarshal("", &cfg)
	cfg.TypeMap, err = codegen.ModelMapToTypeInfo(cfg.Models)
	dir := filepath.Dir(filename)
	if !filepath.IsAbs(cfg.OpenAPISchema) {
		cfg.OpenAPISchema = filepath.Join(dir, cfg.OpenAPISchema)
	}

	if cfg.Target == "" {
		// default target-path for codegen is one dir above
		// the schema.
		cfg.Target = dir
	} else if !filepath.IsAbs(cfg.Target) {
		cfg.Target = filepath.Join(dir, cfg.Target)
	}

	if cfg.Package == "" {
		pkgPath, err := code.PkgPath(code.DefaultConfig, cfg.Target)
		if err != nil {
			return err
		}
		cfg.Package = pkgPath
	}
	return nil
}

// LoadGraph loads the schema package from the given schema path,
// and constructs a *gen.Graph.
func LoadGraph(schemaPath string, cfg *codegen.Config) (*codegen.Graph, error) {
	spec, err := loadSwagger(schemaPath)
	if err != nil {
		return nil, err
	}

	return codegen.NewGraph(cfg, spec)
}

func Generate(schemaPath string, cfg *codegen.Config) error {
	undo, err := codegen.PrepareEnv(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = undo()
		}
	}()
	return generate(schemaPath, cfg)
}

// generate loads the given schema and run codegen.
func generate(schemaPath string, cfg *codegen.Config) error {
	graph, err := LoadGraph(schemaPath, cfg)
	if err != nil {
		if err := mayRecover(err, schemaPath, cfg); err != nil {
			return err
		}
		if graph, err = LoadGraph(schemaPath, cfg); err != nil {
			return err
		}
	}
	if err := normalizePkg(cfg); err != nil {
		return err
	}
	return graph.Gen()
}

func mayRecover(err error, schemaPath string, cfg *codegen.Config) error {
	if !errors.As(err, &packages.Error{}) && !helper.IsBuildError(err) {
		return err
	}
	// If the build error comes from the schema package.
	if err := helper.CheckDir(schemaPath); err != nil {
		return fmt.Errorf("schema failure: %w", err)
	}
	return nil
}

func normalizePkg(c *codegen.Config) error {
	base := path.Base(c.Package)
	if strings.ContainsRune(base, '-') {
		base = strings.ReplaceAll(base, "-", "_")
		c.Package = path.Join(path.Dir(c.Package), base)
	}
	if !token.IsIdentifier(base) {
		return fmt.Errorf("invalid package identifier: %q", base)
	}
	return nil
}
