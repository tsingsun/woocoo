package oasgen

import (
	"embed"
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/gen"
	"path/filepath"
)

type (
	NodeTemplate struct {
		Name           string
		Format         func(*Tag) string
		ExtendPatterns []string
		Skip           func(*Graph) bool
	}

	GraphTemplate struct {
		Name           string
		Format         string
		ExtendPatterns []string
		Skip           func(*Graph) bool
	}
)

var (
	Templates = []NodeTemplate{
		{
			Name:   "tag",
			Format: pkgf("tag_%s.go"),
		},
		{
			Name:   "api",
			Format: pkgf("api_%s.go"),
			Skip:   SkipGenServer,
		},
	}
	GraphTemplates = []GraphTemplate{
		{
			Name:   "interface",
			Format: "interface.go",
			Skip:   SkipGenClient,
		},
		{
			Name:   "schema",
			Format: "model.go",
		},
		{
			Name:   "handler",
			Format: "handler.go",
			Skip:   SkipGenClient,
		},
		{
			Name:   "validator",
			Format: "validator.go",
			Skip:   SkipGenClient,
		},
		{
			Name:   "client",
			Format: "client.go",
			Skip:   SkipGenServer,
		},
	}
	partialPatterns = [...]string{
		"additional/*",
	}
	// templates holds the Go templates for the code generation.
	templates *gen.Template
	//go:embed template/*
	templateDir embed.FS
	// importPkg are the import packages used for code generation.
	// Extended by the function below on generation initialization.
	importPkg = map[string]string{
		"context": "context",
		"errors":  "errors",
		"fmt":     "fmt",
		"math":    "math",
		"strings": "strings",
		"time":    "time",
		"regexp":  "regexp",
	}
)

func initTemplates() {
	tpkgs := make(map[string]string)
	templates = gen.ParseT("templates", templateDir, funcs, "template/*.tmpl")
	tpkgs = gen.InitTemplates(templates, "import", Tag{Config: &Config{}})
	for k, v := range tpkgs {
		importPkg[k] = v
	}
}

func pkgf(s string) func(t *Tag) string {
	return func(t *Tag) string { return fmt.Sprintf(s, t.Spec.Name) }
}

func SkipGenClient(graph *Graph) bool {
	return graph.GenClient
}

func SkipGenServer(graph *Graph) bool {
	return !graph.GenClient
}

// match reports if the given name matches the extended pattern.
func match(patterns []string, name string) bool {
	for _, pat := range patterns {
		matched, _ := filepath.Match(pat, name)
		if matched {
			return true
		}
	}
	return false
}

// NewTemplate creates an empty template with the standard codegen functions.
func NewTemplate(name string) *gen.Template {
	t := gen.NewTemplate(name)
	return t.Funcs(funcs)
}
