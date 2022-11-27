package project

import (
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
)

type (
	Config struct {
		Package        string `json:"package"`
		Target         string `json:"target,omitempty"`
		Templates      []*helper.Template
		Header         string   `json:"header,omitempty"`
		Modules        []string `json:"modules,omitempty"`
		SupportModules []string `json:"supportModules"`
	}
	Graph struct {
		*Config
		Mods []string
	}

	Generator interface {
		// Generate generates the ent artifacts for the given graph.
		Generate(*Graph) error
	}

	// The GenerateFunc type is an adapter to allow the use of ordinary
	// function as Generator. If f is a function with the appropriate signature,
	// GenerateFunc(f) is a Generator that calls f.
	GenerateFunc func(*Graph) error
)

func (f GenerateFunc) Generate(g *Graph) error {
	return f(g)
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

// Gen generates the artifacts for the graph.
func (g *Graph) Gen(gf GenerateFunc) error {
	var gen Generator = gf
	return gen.Generate(g)
}

func (g *Graph) HasModule(name string) bool {
	for _, m := range g.Modules {
		if m == name {
			return true
		}
	}
	return false
}

func (g *Graph) HasStore() bool {
	if g.HasModule("redis") {
		return true
	}
	if g.HasModule("mysql") {
		return true
	}
	return false
}
