package oasgen

import (
	"bytes"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/gen"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template/parse"
)

var defaultTagName = "service"

type (
	// The Config holds the global codegen configuration to be
	// shared between all generated nodes.
	Config struct {
		OpenAPISchema string `json:"spec,omitempty"`
		Package       string `json:"package"`
		Target        string `json:"target,omitempty"`
		Header        string `json:"header,omitempty"`
		// Templates specifies a list of alternative templates to execute or
		// to override the default. If nil, the default template is used.
		//
		// Note that, additional templates are executed on the Graph object and
		// the execution output is stored in a file derived by the template name.
		Templates []*gen.Template
		// Hooks holds an optional list of Hooks to apply on the graph before/after the code-generation.
		Hooks          []gen.Hook
		GeneratedHooks []gen.GeneratedHook
		Models         map[string]*ModelMap `json:"models,omitempty"`
		TypeMap        map[string]*code.TypeInfo
		// Schemas is the list of all schemas reference in the spec.
		Schemas []*Schema
		schemas map[string]*Schema
		// GenClient is the flag for client side code generation
		GenClient bool
	}

	ModelMap struct {
		Model string `json:"model,omitempty"`
	}

	Graph struct {
		*Config
		Nodes []*Tag
		nodes map[string]*Tag
		Spec  *openapi3.T
	}
)

func (g *Graph) Name() string {
	return "OpenAPI-Generator"
}

func (g *Graph) Templates() []*gen.Template {
	return g.Config.Templates
}

func (g *Graph) Hooks() []gen.Hook {
	return g.Config.Hooks
}

func (g *Graph) GeneratedHooks() []gen.GeneratedHook {
	return g.Config.GeneratedHooks
}

func (c *Config) Imports() []string {
	var imp []string
	for _, t := range c.TypeMap {
		if t.PkgPath != c.Package {
			imp = append(imp, t.PkgPath)
		}
	}
	return imp
}

func (c *Config) AddTypeMap(key string, t *code.TypeInfo) {
	if c.TypeMap == nil {
		c.TypeMap = make(map[string]*code.TypeInfo)
	}
	if _, ok := c.TypeMap[key]; !ok {
		c.TypeMap[key] = t
	}
}

func (c *Config) AddSchema(ref string, schema *Schema) {
	if c.schemas == nil {
		c.schemas = make(map[string]*Schema)
	}
	if _, ok := c.schemas[ref]; ok {
		return
	}

	c.schemas[ref] = schema
	c.Schemas = append(c.Schemas, schema)
}

// NewGraph creates a new Graph for the code generation from the given Spec definitions.
// It fails if one of the schemas is invalid.
func NewGraph(c *Config, schema *openapi3.T) (g *Graph, err error) {
	defer gen.CatchGraphError(&err)
	g = &Graph{
		Config: c,
		Nodes:  make([]*Tag, 0, schema.Paths.Len()),
		nodes:  make(map[string]*Tag),
		Spec:   schema,
	}
	// gen models
	g.addModels(schema)
	// gen operations
	g.addNode(schema)
	return
}

// Gen generates the artifacts for the graph.
func (g *Graph) Gen() error {
	return gen.ExecGen(generate, g)
}

// generate is the default Generator implementation.
func generate(gg gen.Extension) error {
	g := gg.(*Graph)
	var (
		assets   gen.Assets
		external []GraphTemplate
	)
	templates, external = g.templates()
	pkg := g.Package
	assets.AddDir(filepath.Join(g.Config.Target))
	for _, n := range g.Nodes {
		for _, tmpl := range Templates {
			if tmpl.Skip != nil && tmpl.Skip(g) {
				continue
			}
			b := bytes.NewBuffer(nil)
			if err := templates.ExecuteTemplate(b, tmpl.Name, n); err != nil {
				return fmt.Errorf("execute template %q: %w", tmpl.Name, err)
			}
			assets.Add(filepath.Join(g.Config.Target, tmpl.Format(n)), b.Bytes())
		}
	}
	g.Package = pkg
	for _, tmpl := range append(GraphTemplates, external...) {
		if tmpl.Skip != nil && tmpl.Skip(g) {
			continue
		}
		if dir := filepath.Dir(tmpl.Format); dir != "." {
			assets.AddDir(filepath.Join(g.Config.Target, dir))
		}
		b := bytes.NewBuffer(nil)
		if err := templates.ExecuteTemplate(b, tmpl.Name, g); err != nil {
			return fmt.Errorf("execute template %q: %w", tmpl.Name, err)
		}
		assets.Add(filepath.Join(g.Config.Target, tmpl.Format), b.Bytes())
	}
	// Write and Format Assets only if template execution
	// finished successfully.
	if err := assets.Write(); err != nil {
		return err
	}
	// We can't run "imports" on files when the state is not completed.
	// Because, "goimports" will drop undefined package. Therefore, it
	// is suspended to the end of the writing.
	return assets.Format()
}

// templates returns the Template to execute on the Graph,
// and a list of optional external templates if provided.
func (g *Graph) templates() (*gen.Template, []GraphTemplate) {
	initTemplates()
	var (
		roots = make(map[string]struct{})
	)
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
				// If the template does not override or extend one of
				// the builtin templates, generate it in a new file.
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

func (g *Graph) addTag(schema *openapi3.T) {
	for _, t := range schema.Tags {
		tv := &Tag{Config: g.Config, Name: t.Name, Spec: t}
		g.Nodes = append(g.Nodes, tv)
		g.nodes[t.Name] = tv
	}
	if len(g.Nodes) == 0 {
		g.Nodes = append(g.Nodes, &Tag{
			Config: g.Config,
			Spec: &openapi3.Tag{
				Name: defaultTagName,
			},
		},
		)
		g.nodes[defaultTagName] = g.Nodes[0]
	}
}
func (g *Graph) addModels(schema *openapi3.T) {
	genComponentSchemas(g.Config, schema)
	var sc []*Schema
	// map to list
	for _, s := range g.schemas {
		sc = append(sc, s)
	}
	sort.Slice(sc, func(i, j int) bool {
		return sc[i].Name < sc[j].Name
	})
	g.Schemas = sc
}

func (g *Graph) addNode(schema *openapi3.T) {
	g.addTag(schema)
	if schema.Paths.Len() == 0 {
		return
	}
	ops := genOperation(g.Config, schema)
	for _, node := range g.Nodes {
		t := g.findTag(node.Name)
		if t == nil {
			panic(fmt.Sprintf("tag %s not found", node.Name))
		}
		for _, op := range ops {
			if op.Group == node.Name {
				node.Operations = append(node.Operations, op)
			}
		}
	}
	for _, sch := range g.Schemas {
		g.modelXmlTag(sch)
	}
}

// find Tag by name, name can be empty
func (g *Graph) findTag(name string) *Tag {
	for _, t := range g.Nodes {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (g *Graph) modelXmlTag(schema *Schema) {
	if x := schema.Spec.Value.XML; x != nil {
		for i, tag := range schema.StructTags {
			if strings.HasPrefix(tag, "xml") {
				var t string
				if x.Prefix != "" {
					t += x.Prefix + ":"
				}
				t += x.Name
				if x.Attribute {
					t += ",attr"
				}
				schema.StructTags[i] = fmt.Sprintf("xml:\"%s\"", t)
			}
		}
	}
	for _, property := range schema.properties {
		g.modelXmlTag(property)
	}
}

// PrepareEnv makes sure the generated directory (environment)
// is suitable for loading the `ent` package (avoid cyclic imports).
func PrepareEnv(c *Config) (undo func() error, err error) {
	var (
		nop  = func() error { return nil }
		path = filepath.Join(c.Target, "runtime.go")
	)
	out, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nop, nil
		}
		return nil, err
	}
	fi, err := parser.ParseFile(token.NewFileSet(), path, out, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	// Targeted package doesn't import the Spec.
	if len(fi.Imports) == 0 {
		return nop, nil
	}
	if err := os.WriteFile(path, append([]byte("// +build tools\n"), out...), 0644); err != nil {
		return nil, err
	}
	return func() error { return os.WriteFile(path, out, 0644) }, nil
}

func extendExisting(name string) bool {
	if match(partialPatterns[:], name) {
		return true
	}
	for _, t := range Templates {
		if match(t.ExtendPatterns, name) {
			return true
		}
	}
	for _, t := range GraphTemplates {
		if match(t.ExtendPatterns, name) {
			return true
		}
	}
	return false
}
