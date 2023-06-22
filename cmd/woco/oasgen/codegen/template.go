package codegen

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
)

type (
	NodeTemplate struct {
		Name           string
		Format         func(*Tag) string
		ExtendPatterns []string
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
			Format: pkgf("%s_tag.go"),
		},
	}
	GraphTemplates = []GraphTemplate{
		{
			Name:   "interface",
			Format: "interface.go",
		},
		{
			Name:   "schema",
			Format: "model.go",
		},
		{
			Name:   "server",
			Format: "server/server.go",
		},
		{
			Name:   "validator",
			Format: "server/validator.go",
		},
	}
	partialPatterns = [...]string{
		"additional/*",
	}
	// templates holds the Go templates for the code generation.
	templates *helper.Template
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
	templates = helper.MustParse(NewTemplate("templates").
		ParseFS(templateDir, "template/*.tmpl"))
	b := bytes.NewBuffer([]byte("package main\n"))
	helper.CheckGraphError(templates.ExecuteTemplate(b, "import", Tag{Config: &Config{}}), "load imports")
	f, err := parser.ParseFile(token.NewFileSet(), "", b, parser.ImportsOnly)
	helper.CheckGraphError(err, "parse imports")
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		helper.CheckGraphError(err, "unquote import path")
		importPkg[filepath.Base(path)] = path
	}
}

func pkgf(s string) func(t *Tag) string {
	return func(t *Tag) string { return fmt.Sprintf(s, t.Spec.Name) }
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
func NewTemplate(name string) *helper.Template {
	t := helper.NewTemplate(name)
	return t.Funcs(funcs)
}
