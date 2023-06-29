package project

import (
	"embed"
	"github.com/tsingsun/woocoo/cmd/woco/gen"
)

type (
	GraphTemplate struct {
		Name           string
		Format         string
		ExtendPatterns []string
	}
)

var (
	// templates holds the Go templates for the code generation.
	templates *gen.Template
	//go:embed template/*
	templateDir embed.FS
	importPkg   = map[string]string{
		"context": "context",
		"errors":  "errors",
		"fmt":     "fmt",
		"math":    "math",
		"strings": "strings",
		"time":    "time",
		"regexp":  "regexp",
	}
)

var (
	GraphTemplates = []GraphTemplate{
		{
			Name:   "config",
			Format: "cmd/etc/app.yaml",
		},
		{
			Name:   "main",
			Format: "cmd/main.go",
		},
		{
			Name:   "mod",
			Format: "go.mod",
		},
		{
			Name:   "makefile",
			Format: "Makefile",
		},
		{
			Name:   "readme",
			Format: "README.md",
		},
		{
			Name:   "gitignore",
			Format: ".gitignore",
		},
	}
)

func initTemplates() {
	tpkgs := make(map[string]string)
	templates = gen.ParseT("templates", templateDir, nil, "template/*.tmpl", "template/*/*.tmpl")
	tpkgs = gen.InitTemplates(templates, "import", Graph{Config: &Config{}})
	for k, v := range tpkgs {
		importPkg[k] = v
	}
}

func prepareTemplates(g *Graph) *gen.Template {
	initTemplates()
	return templates
}
