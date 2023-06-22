package project

import (
	"bytes"
	"embed"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
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
	templates *helper.Template
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
	templates = helper.MustParse(helper.NewTemplate("templates").
		ParseFS(templateDir, "template/*.tmpl", "template/*/*.tmpl"))
	b := bytes.NewBuffer([]byte("package main\n"))
	helper.CheckGraphError(templates.ExecuteTemplate(b, "import", Graph{Config: &Config{}}), "load imports")
	f, err := parser.ParseFile(token.NewFileSet(), "", b, parser.ImportsOnly)
	helper.CheckGraphError(err, "parse imports")
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		helper.CheckGraphError(err, "unquote import path")
		importPkg[filepath.Base(path)] = path
	}
}

func prepareTemplates(g *Graph) *helper.Template {
	initTemplates()
	return templates
}
