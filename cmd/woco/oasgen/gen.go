package oasgen

import (
	"context"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/gen"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/codegen"
	"github.com/tsingsun/woocoo/pkg/conf"
	"golang.org/x/tools/go/packages"
	"net/url"
	"path/filepath"
)

type Option func(*codegen.Config) error

// TemplateDir parses the template definitions from the files in the directory
// and associates the resulting templates with codegen templates.
func TemplateDir(path string) Option {
	return templateOption(func(t *gen.Template) (*gen.Template, error) {
		return t.ParseDir(path)
	})
}

// TemplateFiles parses the named files and associates the resulting templates
// with codegen templates.
func TemplateFiles(filenames ...string) Option {
	return templateOption(func(t *gen.Template) (*gen.Template, error) {
		return t.ParseFiles(filenames...)
	})
}

func loadSwagger(path string) (oas *openapi3.T, err error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	if u.Scheme != "" && u.Host != "" {
		oas, err = loader.LoadFromURI(u)
	} else {
		oas, err = loader.LoadFromFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load swagger file: %w", err)
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

	if cfg.Target == "" {
		// default target-path for codegen is one dir above
		// the schema.
		cfg.Target = dir
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

func Generate(schemaPath string, cfg *codegen.Config, options ...Option) error {
	for _, opt := range options {
		if err := opt(cfg); err != nil {
			return err
		}
	}
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

// templateOption ensures the template instantiate
// once for config and execute the given Option.
func templateOption(next func(t *gen.Template) (*gen.Template, error)) Option {
	return func(cfg *codegen.Config) (err error) {
		tmpl, err := next(codegen.NewTemplate("external"))
		if err != nil {
			return err
		}
		cfg.Templates = append(cfg.Templates, tmpl)
		return nil
	}
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
	np, err := helper.NormalizePkg(cfg.Package)
	if err != nil {
		return err
	}
	cfg.Package = np
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
