package codegen

import (
	"bytes"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"go/parser"
	"go/token"
	"golang.org/x/tools/imports"
	"log"
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
		Templates []*Template
		// Hooks holds an optional list of Hooks to apply on the graph before/after the code-generation.
		Hooks []Hook

		Models  map[string]*ModelMap `json:"models,omitempty"`
		TypeMap map[string]*code.TypeInfo
		// Schemas is the list of all schemas reference in the spec.
		Schemas []*Schema
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

	Generator interface {
		// Generate generates the ent artifacts for the given graph.
		Generate(*Graph) error
	}

	// The GenerateFunc type is an adapter to allow the use of ordinary
	// function as Generator. If f is a function with the appropriate signature,
	// GenerateFunc(f) is a Generator that calls f.
	GenerateFunc func(*Graph) error

	// Hook defines the "generate middleware". A function that gets a Generator
	// and returns a Generator. For example:
	//
	//	hook := func(next gen.Generator) gen.Generator {
	//		return gen.GenerateFunc(func(g *Graph) error {
	//			fmt.Println("Graph:", g)
	//			return next.Generate(g)
	//		})
	//	}
	//
	Hook func(Generator) Generator
)

func (f GenerateFunc) Generate(g *Graph) error {
	return f(g)
}

func (c Config) Imports() []string {
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

// NewGraph creates a new Graph for the code generation from the given Spec definitions.
// It fails if one of the schemas is invalid.
func NewGraph(c *Config, schema *openapi3.T) (g *Graph, err error) {
	defer catch(&err)
	g = &Graph{
		Config: c,
		Nodes:  make([]*Tag, 0, len(schema.Paths)),
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
	var gen Generator = GenerateFunc(generate)
	for i := len(g.Hooks) - 1; i >= 0; i-- {
		gen = g.Hooks[i](gen)
	}
	return gen.Generate(g)
}

// generate is the default Generator implementation.
func generate(g *Graph) error {
	var (
		assets   assets
		external []GraphTemplate
	)
	templates, external = g.templates()
	pkg := g.Package
	assets.addDir(filepath.Join(g.Config.Target))
	for _, n := range g.Nodes {
		//npkg := n.PackageDir()
		//if npkg == "" {
		//	npkg = pkg
		//}
		//npkg = strings.ToLower(n.PackageDir())
		for _, tmpl := range Templates {
			b := bytes.NewBuffer(nil)
			//n.Package = npkg
			if err := templates.ExecuteTemplate(b, tmpl.Name, n); err != nil {
				return fmt.Errorf("execute template %q: %w", tmpl.Name, err)
			}
			assets.add(filepath.Join(g.Config.Target, tmpl.Format(n)), b.Bytes())
		}
	}
	g.Package = pkg
	for _, tmpl := range append(GraphTemplates, external...) {
		if tmpl.Skip != nil && tmpl.Skip(g) {
			continue
		}
		if dir := filepath.Dir(tmpl.Format); dir != "." {
			assets.addDir(filepath.Join(g.Config.Target, dir))
		}
		b := bytes.NewBuffer(nil)
		if err := templates.ExecuteTemplate(b, tmpl.Name, g); err != nil {
			return fmt.Errorf("execute template %q: %w", tmpl.Name, err)
		}
		assets.add(filepath.Join(g.Config.Target, tmpl.Format), b.Bytes())
	}
	// Write and format assets only if template execution
	// finished successfully.
	if err := assets.write(); err != nil {
		return err
	}
	// cleanup assets that are not needed anymore.
	cleanOldNodes(assets, g.Config.Target)
	// We can't run "imports" on files when the state is not completed.
	// Because, "goimports" will drop undefined package. Therefore, it
	// is suspended to the end of the writing.
	return assets.format()
}

// templates returns the Template to execute on the Graph,
// and a list of optional external templates if provided.
func (g *Graph) templates() (*Template, []GraphTemplate) {
	initTemplates()
	var (
		roots    = make(map[string]struct{})
		helpers  = make(map[string]struct{})
		external = make([]GraphTemplate, 0, len(g.Templates))
	)
	for _, rootT := range g.Templates {
		templates.Funcs(rootT.FuncMap)
		for _, tmpl := range rootT.Templates() {
			if parse.IsEmptyTree(tmpl.Root) {
				continue
			}
			name := tmpl.Name()
			switch {
			// Helper templates can be either global (prefixed with "helper/"),
			// or local, where their names follow the format: "<root-tmpl>/helper/.+").
			case strings.HasPrefix(name, "helper/"):
			case strings.Contains(name, "/helper/"):
				helpers[name] = struct{}{}
			case templates.Lookup(name) == nil && !extendExisting(name):
				// If the template does not override or extend one of
				// the builtin templates, generate it in a new file.
				external = append(external, GraphTemplate{
					Name:   name,
					Format: helper.Snake(name) + ".go",
					Skip:   rootT.condition,
				})
				roots[name] = struct{}{}
			}
			templates = MustParse(templates.AddParseTree(name, tmpl.Tree))
		}
	}
	for name := range helpers {
		root := name[:strings.Index(name, "/helper/")]
		// If the name is prefixed with a name of a root
		// template, we treat it as a local helper template.
		if _, ok := roots[root]; ok {
			continue
		}
		external = append(external, GraphTemplate{
			Name:   name,
			Format: helper.Snake(name) + ".go",
		})
	}
	return templates, external
}

func (g *Graph) addTag(schema *openapi3.T) {
	for _, t := range schema.Tags {
		tv := &Tag{Config: g.Config, Name: t.Name, Spec: t}
		g.Nodes = append(g.Nodes, tv)
		g.nodes[t.Name] = tv
	}
	if len(g.Nodes) == 0 {
		g.Nodes = append(g.Nodes, &Tag{Name: defaultTagName})
		g.nodes[defaultTagName] = g.Nodes[0]
	}
}
func (g *Graph) addModels(schema *openapi3.T) {
	sc := genComponentSchemas(g.Config, schema)
	sort.Slice(sc, func(i, j int) bool {
		return sc[i].Name < sc[j].Name
	})
	g.Schemas = sc
}

func (g *Graph) addNode(schema *openapi3.T) {
	g.addTag(schema)
	if len(schema.Paths) == 0 {
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
}

// find Tag by name
func (g *Graph) findTag(name string) *Tag {
	if name == "" {
		name = defaultTagName
	}
	for _, t := range g.Nodes {
		if t.Name == name {
			return t
		}
	}
	return nil
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

// cleanOldNodes removes all files that were generated
// for nodes that were removed from the Spec.
func cleanOldNodes(assets assets, target string) {
	d, err := os.ReadDir(target)
	if err != nil {
		return
	}
	// Find deleted nodes by selecting one generated
	// file from standard templates (<T>_query.go).
	var deleted []*Tag
	for _, f := range d {
		if !strings.HasSuffix(f.Name(), "_query.go") {
			continue
		}
		return
		//typ := &Operation{}
		//path := filepath.Join(target, typ.PackageDir())
		//if _, ok := assets.dirs[path]; ok {
		//	continue
		//}
		//// If it is a node, it must have a model file and a dir (e.g. ent/t.go, ent/t).
		//_, err1 := os.Stat(path + ".go")
		//f2, err2 := os.Stat(path)
		//if err1 == nil && err2 == nil && f2.IsDir() {
		//	deleted = append(deleted, typ)
		//}
	}
	for _, typ := range deleted {
		for _, t := range Templates {
			err := os.Remove(filepath.Join(target, t.Format(typ)))
			if err != nil && !os.IsNotExist(err) {
				log.Printf("remove old file %s: %s\n", filepath.Join(target, t.Format(typ)), err)
			}
		}
		err := os.Remove(filepath.Join(target, typ.PackageDir()))
		if err != nil && !os.IsNotExist(err) {
			log.Printf("remove old dir %s: %s\n", filepath.Join(target, typ.PackageDir()), err)
		}
	}
}

type assets struct {
	dirs  map[string]struct{}
	files map[string][]byte
}

func (a *assets) add(path string, content []byte) {
	if a.files == nil {
		a.files = make(map[string][]byte)
	}
	a.files[path] = content
}

func (a *assets) addDir(path string) {
	if a.dirs == nil {
		a.dirs = make(map[string]struct{})
	}
	a.dirs[path] = struct{}{}
}

// write files and dirs in the assets.
func (a assets) write() error {
	for dir := range a.dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("create dir %q: %w", dir, err)
		}
	}
	for path, content := range a.files {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("write file %q: %w", path, err)
		}
	}
	return nil
}

// format runs "goimports" on all assets.
func (a assets) format() error {
	for path, content := range a.files {
		src, err := imports.Process(path, content, nil)
		if err != nil {
			return fmt.Errorf("format file %s: %w", path, err)
		}
		if err := os.WriteFile(path, src, 0644); err != nil {
			return fmt.Errorf("write file %s: %w", path, err)
		}
	}
	return nil
}

// expect panics if the condition is false.
func expect(cond bool, msg string, args ...any) {
	if !cond {
		panic(graphError{fmt.Sprintf(msg, args...)})
	}
}

func check(err error, msg string, args ...any) {
	if err != nil {
		args = append(args, err)
		panic(graphError{fmt.Sprintf(msg+": %s", args...)})
	}
}

type graphError struct {
	msg string
}

func (p graphError) Error() string { return fmt.Sprintf("entc/gen: %s", p.msg) }

func catch(err *error) {
	if e := recover(); e != nil {
		gerr, ok := e.(graphError)
		if !ok {
			panic(e)
		}
		*err = gerr
	}
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
