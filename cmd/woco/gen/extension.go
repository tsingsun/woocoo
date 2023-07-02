package gen

// Extension is a plugin that can be used to extend the code generation.
// every code generation has a default extension.
type Extension interface {
	// Name is the plugin name
	Name() string
	// Templates specifies a list of alternative templates
	// to execute or to override the default.
	Templates() []*Template

	// Hooks holds an optional list of Hooks to apply
	// on the graph before the code-generation.
	Hooks() []Hook
	// GeneratedHooks holds an optional list of Hooks to apply after the code-generation.
	// plugin extension will run first
	GeneratedHooks() []GeneratedHook
}

type Generator interface {
	Generate(Extension) error
}

// The GenerateFunc type is an adapter to allow the use of ordinary
// function as Generator. If f is a function with the appropriate signature,
// GenerateFunc(f) is a Generator that calls f.
type GenerateFunc func(Extension) error

func (g GenerateFunc) Generate(graph Extension) error {
	return g(graph)
}

type Hook func(Generator) Generator
type GeneratedHook func(Extension) error

func ExecGen(entry GenerateFunc, g Extension) error {
	var gg Generator = GenerateFunc(entry)
	entryex := g.(Extension)
	hooks := entryex.Hooks()
	for i := len(hooks) - 1; i >= 0; i-- {
		gg = hooks[i](gg)
	}
	err := gg.Generate(g)
	if err != nil {
		return err
	}
	generatedHooks := entryex.GeneratedHooks()
	for _, hook := range generatedHooks {
		err = hook(entryex)
		if err != nil {
			return err
		}
	}
	return nil
}
