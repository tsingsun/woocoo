package codegen

import (
	"bytes"
	"embed"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
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
	}
	partialPatterns = [...]string{
		"additional/*",
	}
	// templates holds the Go templates for the code generation.
	templates *Template
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
	}
)

func initTemplates() {
	templates = MustParse(NewTemplate("templates").
		ParseFS(templateDir, "template/*.tmpl"))
	b := bytes.NewBuffer([]byte("package main\n"))
	check(templates.ExecuteTemplate(b, "import", Tag{Config: &Config{}}), "load imports")
	f, err := parser.ParseFile(token.NewFileSet(), "", b, parser.ImportsOnly)
	check(err, "parse imports")
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		check(err, "unquote import path")
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

// Template wraps the standard template.Template to
// provide additional functionality for ent extensions.
type Template struct {
	*template.Template
	FuncMap   template.FuncMap
	condition func(*Graph) bool
}

// NewTemplate creates an empty template with the standard codegen functions.
func NewTemplate(name string) *Template {
	t := &Template{Template: template.New(name)}
	return t.Funcs(Funcs)
}

// Funcs merges the given funcMap with the template functions.
func (t *Template) Funcs(funcMap template.FuncMap) *Template {
	t.Template.Funcs(funcMap)
	if t.FuncMap == nil {
		t.FuncMap = template.FuncMap{}
	}
	for name, f := range funcMap {
		if _, ok := t.FuncMap[name]; !ok {
			t.FuncMap[name] = f
		}
	}
	return t
}

// SkipIf allows registering a function to determine if the template needs to be skipped or not.
func (t *Template) SkipIf(cond func(*Graph) bool) *Template {
	t.condition = cond
	return t
}

// Parse parses text as a template body for t.
func (t *Template) Parse(text string) (*Template, error) {
	if _, err := t.Template.Parse(text); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseFiles parses a list of files as templates and associate them with t.
// Each file can be a standalone template.
func (t *Template) ParseFiles(filenames ...string) (*Template, error) {
	if _, err := t.Template.ParseFiles(filenames...); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseGlob parses the files that match the given pattern as templates and
// associate them with t.
func (t *Template) ParseGlob(pattern string) (*Template, error) {
	if _, err := t.Template.ParseGlob(pattern); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseDir walks on the given dir path and parses the given matches with aren't Go files.
func (t *Template) ParseDir(path string) (*Template, error) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk path %s: %w", path, err)
		}
		if info.IsDir() || strings.HasSuffix(path, ".go") {
			return nil
		}
		_, err = t.ParseFiles(path)
		return err
	})
	return t, err
}

// ParseFS is like ParseFiles or ParseGlob but reads from the file system fsys
// instead of the host operating system's file system.
func (t *Template) ParseFS(fsys fs.FS, patterns ...string) (*Template, error) {
	if _, err := t.Template.ParseFS(fsys, patterns...); err != nil {
		return nil, err
	}
	return t, nil
}

// AddParseTree adds the given parse tree to the template.
func (t *Template) AddParseTree(name string, tree *parse.Tree) (*Template, error) {
	if _, err := t.Template.AddParseTree(name, tree); err != nil {
		return nil, err
	}
	return t, nil
}

// MustParse is a helper that wraps a call to a function returning (*Template, error)
// and panics if the error is non-nil.
func MustParse(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}
