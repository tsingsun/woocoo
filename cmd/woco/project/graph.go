package project

import (
	"bytes"
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/gen"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"path/filepath"
	"strings"
	"text/template/parse"
)

type (
	Option func(*Config) error

	Config struct {
		Package   string `json:"package"`
		Target    string `json:"target,omitempty"`
		Templates []*gen.Template
		Header    string   `json:"header,omitempty"`
		Modules   []string `json:"modules,omitempty"`
		// Hooks hold an optional list of Hooks to apply on the graph before/after the code-generation.
		Hooks          []gen.Hook
		GeneratedHooks []gen.GeneratedHook
		SkipModTidy    bool
	}
	Graph struct {
		*Config
		Mods []string
	}
)

func SkipModTidy() Option {
	return func(config *Config) error {
		config.SkipModTidy = true
		return nil
	}
}

func Extensions(extensions ...gen.Extension) Option {
	return func(config *Config) error {
		for _, ex := range extensions {
			config.Hooks = append(config.Hooks, ex.Hooks()...)
			config.Templates = append(config.Templates, ex.Templates()...)
			config.GeneratedHooks = append(config.GeneratedHooks, ex.GeneratedHooks()...)
		}
		return nil
	}
}

func (c Config) Imports() []string {
	var imp []string
	return imp
}

func NewGraph(c *Config) (g *Graph, err error) {
	g = &Graph{
		Config: c,
	}
	for _, module := range g.Modules {
		switch module {
		default:
		}
	}
	return g, nil
}

// Name implement gen.Graph interface
func (*Graph) Name() string {
	return "ProjectInit"
}

// Gen generates the artifacts for the graph.
func (g *Graph) Gen() error {
	return gen.ExecGen(generate, g)
}

func (g *Graph) Hooks() []gen.Hook {
	return nil
}

func (g *Graph) GeneratedHooks() []gen.GeneratedHook {
	return g.Config.GeneratedHooks
}

func (g *Graph) Templates() []*gen.Template {
	return g.Config.Templates
}

func (g *Graph) HasModule(name string) bool {
	for _, m := range g.Modules {
		if m == name {
			return true
		}
	}
	return false
}

func Generate(cfg *Config, opts ...Option) error {
	err := loadConfig(cfg)
	if err != nil {
		return err
	}
	np, err := helper.NormalizePkg(cfg.Package)
	if err != nil {
		return err
	}

	cfg.Package = np
	cfg.GeneratedHooks = append(cfg.GeneratedHooks, func(extension gen.Extension) error {
		if cfg.SkipModTidy {
			return nil
		}
		return gen.RunCmd(cfg.Target, "go", "mod", "tidy")
	})

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return err
		}
	}
	lg, err := NewGraph(cfg)
	if err != nil {
		return err
	}
	return lg.Gen()
}

// Actually execute the generated code.
func generate(gg gen.Extension) error {
	g := gg.(*Graph)
	var (
		assets   gen.Assets
		external []GraphTemplate
	)
	templates, external = g.templates()
	//pkg := g.Package
	assets.AddDir(filepath.Join(g.Config.Target))
	for _, tmpl := range append(GraphTemplates, external...) {
		b := bytes.NewBuffer(nil)
		if dir := filepath.Dir(tmpl.Format); dir != "." {
			assets.AddDir(filepath.Join(g.Config.Target, dir))
		}
		if err := templates.ExecuteTemplate(b, tmpl.Name, g); err != nil {
			return fmt.Errorf("execute template %q: %w", tmpl.Name, err)
		}
		assets.Add(filepath.Join(g.Target, tmpl.Format), b.Bytes())
	}

	// Write and Format Assets only if template execution
	// finished successfully.
	if err := assets.Write(); err != nil {
		return err
	}
	return assets.Format()
}

func loadConfig(cfg *Config) error {
	if cfg.Package == "" {
		pkgPath, err := code.PkgPath(code.DefaultConfig, cfg.Target)
		if err != nil {
			return err
		}
		cfg.Package = pkgPath
	}
	return nil
}

func (g *Graph) templates() (*gen.Template, []GraphTemplate) {
	initTemplates()
	var (
		roots = make(map[string]struct{})
	)
	p := &code.Packages{}
	for _, v := range importPkg {
		if !strings.ContainsAny(v, "/") {
			continue
		}
		mpkg := p.Load(v)
		if mpkg.Module != nil {
			g.Mods = append(g.Mods, mpkg.Module.Path+" "+mpkg.Module.Version)
		}
	}
	gt := make([]GraphTemplate, 0, len(g.Config.Templates))
	for _, rootT := range g.Config.Templates {
		templates.Funcs(rootT.FuncMap)
		for _, tpl := range rootT.Templates() {
			if parse.IsEmptyTree(tpl.Root) {
				continue
			}
			name := tpl.Name()
			switch {
			case templates.Lookup(name) == nil && !extendExisting(name):
				format := helper.Snake(name)
				if filepath.Ext(name) == "" {
					format += ".go"
				}
				gt = append(gt, GraphTemplate{
					Name:   name,
					Format: format,
				})
				roots[name] = struct{}{}
			}
			templates = gen.MustParse(templates.AddParseTree(name, tpl.Tree))
		}
	}
	return templates, gt
}

func extendExisting(name string) bool {
	for _, t := range GraphTemplates {
		if helper.PathMatch(t.ExtendPatterns, name) {
			return true
		}
	}
	return false
}
