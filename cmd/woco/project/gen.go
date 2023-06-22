package project

import (
	"bytes"
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"path/filepath"
	"strings"
)

// loadGraph loads the schema package from the given schema path,
// and constructs a *gen.Graph.
func loadGraph(cfg *Config) (*Graph, error) {
	return NewGraph(cfg)
}

func generate(g *Graph) error {
	var (
		assets helper.Assets
	)
	templates = prepareTemplates(g)
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
	//pkg := g.Package
	assets.AddDir(filepath.Join(g.Config.Target))
	for _, tmpl := range GraphTemplates {
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
	err := assets.Format()
	if err != nil {
		return err
	}
	return assets.ModTidy(g.Target)
}

// generate is the default Generator implementation.
func generateWeb(cfg *Config) error {
	err := loadConfig(cfg)
	if err != nil {
		return err
	}
	np, err := helper.NormalizePkg(cfg.Package)
	if err != nil {
		return err
	}
	cfg.Package = np
	lg, err := loadGraph(cfg)
	if err != nil {
		return err
	}
	return lg.Gen(generate)
}

func loadConfig(cfg *Config) error {
	if cfg.Package == "" {
		pkgPath, err := code.PkgPath(code.DefaultConfig, cfg.Target)
		if err != nil {
			return err
		}
		cfg.Package = pkgPath
	}
	s := make([]string, 0)
	if len(cfg.SupportModules) == 0 {
		cfg.SupportModules = []string{"web", "grpc", "cache", "db", "otel"}
	}
	for _, m := range cfg.Modules {
		if helper.InStrSlice(cfg.SupportModules, m) {
			s = append(s, m)
		} else {
			return fmt.Errorf("unsupported module %q", m)
		}
	}
	cfg.Modules = s
	return nil
}
