package conf

import (
	"bytes"
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Configurable can initial by framework
type Configurable interface {
	// Apply set up property or field value by configuration
	//
	// cnf is the Configuration of the component, and it's sub configuration of root
	// path is the relative path to root,if root is the component,path will be empty
	// Apply also use for lazy load scene
	// notice: if any error in apply process,use panic to expose error,and stop application.
	Apply(cnf *Configuration)
}

const (
	baseDirEnv = "WOOCOO_BASEDIR"
)

var (
	global            AppConfiguration
	defaultConfigFile = filepath.Join("etc", "app.yaml")
)

type (
	// Configuration hold settings of the component.
	Configuration struct {
		opts        options
		parser      *Parser
		Development bool
		root        *Configuration
	}
	// AppConfiguration is the application level configuration,include all of component's configurations
	AppConfiguration struct {
		*Configuration
	}
)

var defaultOptions = options{
	global: true,
}

func init() {
	pwd, err := filepath.Abs(os.Args[0])
	if err != nil {
		panic(err)
	}
	defaultOptions.basedir = filepath.Dir(pwd)
	if bs := os.Getenv(baseDirEnv); bs != "" {
		defaultOptions.basedir = bs
	}
	defaultOptions.localPath = filepath.Join(defaultOptions.basedir, defaultConfigFile)
}

// New create an application configuration instance.
//
// New as global by default,if you want to create a local configuration,set global to false.
// initialization such as:
//
//	cnf := conf.New().Load()
func New(opts ...Option) *Configuration {
	cnf := &Configuration{
		opts:   defaultOptions,
		parser: NewParser(),
	}
	for _, o := range opts {
		o(&cnf.opts)
	}

	if cnf.opts.global {
		cnf.AsGlobal()
	}

	return cnf
}

// NewFromBytes create from byte slice,return with a parser, But you'd better use Load()
func NewFromBytes(b []byte, opts ...Option) *Configuration {
	p, err := NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	return NewFromParse(p, opts...)
}

// NewFromStringMap create from string map
func NewFromStringMap(data map[string]any, opts ...Option) *Configuration {
	p := NewParserFromStringMap(data)
	return NewFromParse(p, opts...)
}

// NewFromParse create from parser
func NewFromParse(parser *Parser, opts ...Option) *Configuration {
	cnf := New(opts...)
	cnf.parser = parser
	return cnf
}

func (c *Configuration) AsGlobal() *Configuration {
	global.Configuration = c
	c.opts.global = true
	return c
}

func (c *Configuration) Load() *Configuration {
	if err := c.loadInternal(); err != nil {
		panic("config load error:" + err.Error())
	}
	c.Development = c.parser.k.Bool("development")
	return c
}

// load configuration,if the RemoteProvider is set,will ignore local configuration
func (c *Configuration) loadInternal() (err error) {
	// if parser is nil, use default local config file
	if c.parser == nil || len(c.parser.AllKeys()) == 0 {
		c.parser, err = NewParserFromFile(c.opts.localPath)
		if err != nil {
			return err
		}
	}

	if c.IsSet("includeFiles") {
		for _, v := range c.StringSlice("includeFiles") {
			path := v
			if !filepath.IsAbs(path) {
				path = filepath.Join(c.GetBaseDir(), v)
			}
			if _, err := os.Stat(path); err != nil {
				panic(fmt.Errorf("config file no exists:%s,cause by:%s", path, err))
			} else {
				c.opts.includeFiles = append(c.opts.includeFiles, path)
			}
		}
	}
	// make sure the "include files" in attach files is not working
	copyifs := make([]string, len(c.opts.includeFiles))
	copy(copyifs, c.opts.includeFiles)
	for _, attach := range copyifs {
		if err = c.parser.k.Load(file.Provider(attach), yaml.Parser()); err != nil {
			return err
		}
	}
	return err
}

// Merge an input config stream,parameter b is YAML stream
//
// types operation:
//   - map[string]any: merge
//   - []any: override
func (c *Configuration) Merge(b []byte) error {
	p, err := NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		return err
	}
	err = c.parser.k.Merge(p.k)
	if err != nil {
		return err
	}
	return nil
}

// GetBaseDir return the application dir
func (c *Configuration) GetBaseDir() string {
	return c.opts.basedir
}

// SetBaseDir return the application dir
func (c *Configuration) SetBaseDir(dir string) {
	c.opts.basedir = dir
}

// Parser return configuration operator
func (c *Configuration) Parser() *Parser {
	return c.parser
}

// ParserOperator return the underlying parser that convert bytes to map
func (c *Configuration) ParserOperator() *koanf.Koanf {
	return c.parser.k
}

// Sub return a new Configuration by a sub node
func (c *Configuration) Sub(path string) *Configuration {
	if path == "" {
		return c
	}
	p, err := c.Parser().Sub(path)
	if err != nil {
		panic(err)
	}
	return &Configuration{
		opts:        c.opts,
		parser:      p,
		Development: c.Development,
		root:        c,
	}
}

// CutFromOperator return a new copied Configuration but replace the parser by koanf operator.
func (c *Configuration) CutFromOperator(kf *koanf.Koanf) *Configuration {
	nf := *c
	nf.parser = &Parser{
		k: kf,
	}
	c.passRoot(&nf)
	return &nf
}

func (c *Configuration) passRoot(sub *Configuration) {
	if c.root != nil {
		sub.root = c.root
	} else {
		sub.root = c
	}
}

// Each iterate the slice path of the Configuration.
func (c *Configuration) Each(path string, cb func(root string, sub *Configuration)) {
	ops := c.ParserOperator().Slices(path)
	for _, op := range ops {
		var root string
		for s := range op.Raw() {
			// first is the node key
			root = s
			break
		}
		kf := op.Cut(root)
		if len(kf.Keys()) == 0 {
			cb(root, c.CutFromOperator(op))
		} else {
			cb(root, c.CutFromOperator(kf))
		}
	}
}

func (c *Configuration) Copy() *Configuration {
	return c.CutFromOperator(c.Parser().k.Copy())
}

// Root return root configuration if it came from Sub method
func (c *Configuration) Root() *Configuration {
	if c.root == nil {
		return c
	}
	return c.root
}

// Namespace indicates the application's namespace
func (c *Configuration) Namespace() string {
	return c.String("namespace")
}

// AppName indicates the application's name
//
// return the path 'appName' value if exists
func (c *Configuration) AppName() string {
	return c.String("appName")
}

// Version indicates the application's sem version
//
// return the path 'version' value, no set return empty
func (c *Configuration) Version() string {
	return c.String("version")
}

// Unmarshal map data of config into a struct, values are merged.
//
// Tags on the fields of the structure must be properly set.
func (c *Configuration) Unmarshal(dst any) (err error) {
	return c.parser.Unmarshal("", dst)
}

// Abs returns the absolute path by the base dir if path is a relative path
func Abs(path string) string { return global.Abs(path) }

func (c *Configuration) Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.GetBaseDir(), path)
}
func Get(path string) any { return global.Get(path) }

func (c *Configuration) Get(key string) any {
	return c.parser.Get(key)
}
func Bool(path string) bool { return global.Bool(path) }

func (c *Configuration) Bool(path string) bool {
	return c.parser.k.Bool(path)
}
func Float64(path string) float64 { return global.Float64(path) }

func (c *Configuration) Float64(path string) float64 {
	return c.parser.k.Float64(path)
}
func Int(path string) int { return global.Int(path) }

func (c *Configuration) Int(path string) int {
	return c.parser.k.Int(path)
}
func IntSlice(path string) []int { return global.IntSlice(path) }

func (c *Configuration) IntSlice(path string) []int {
	return c.parser.k.Ints(path)
}
func String(path string) string { return global.String(path) }

func (c *Configuration) String(path string) string {
	return c.parser.k.String(path)
}
func StringMap(path string) map[string]string { return global.StringMap(path) }

func (c *Configuration) StringMap(path string) map[string]string {
	return c.parser.k.StringMap(path)
}
func StringSlice(path string) []string { return global.StringSlice(path) }

func (c *Configuration) StringSlice(path string) []string {
	return c.parser.k.Strings(path)
}

// Time return time by layout, eg: 2006-01-02 15:04:05
//
// if config is init from a map value of time.Time, the layout will be: `2006-01-02 15:04:05 -0700 MST`
func Time(path string, layout string) time.Time { return global.Time(path, layout) }

func (c *Configuration) Time(path string, layout string) time.Time {
	return c.parser.k.Time(path, layout)
}
func Duration(path string) time.Duration { return global.Duration(path) }

func (c *Configuration) Duration(path string) time.Duration {
	return c.parser.k.Duration(path)
}

// IsSet check if the key is set
func IsSet(path string) bool { return global.IsSet(path) }

// IsSet check if the key is set
func (c *Configuration) IsSet(path string) bool {
	return c.parser.IsSet(path)
}

// AllSettings return all settings
func AllSettings() map[string]any { return global.AllSettings() }

func (c *Configuration) AllSettings() map[string]any {
	return c.parser.k.All()
}

// Join paths
func Join(ps ...string) string {
	if len(ps) == 0 {
		return ps[0]
	}
	if ps[0] == "" {
		return Join(ps[1:]...)
	}
	return strings.Join(ps, KeyDelimiter)
}

// ----------------------------------------------------------------------------

// Global return default(global) Configuration instance
func Global() *AppConfiguration {
	if global.Configuration == nil || global.parser == nil {
		global.Configuration = New().Load()
	}
	return &global
}
