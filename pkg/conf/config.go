package conf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
)

// Configurable can initial by framework
type Configurable interface {
	// Apply set up property or field value by configuration
	//
	// cnf is the Configuration of the component, and it's sub configuration of root
	// path is the relative path to root,if root is the component,path will be empty
	// Apply also use for lazy load scene
	// notice: if any error in apply process,you should choose use panic to expose error,and stop application.
	Apply(cnf *Configuration) error
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
		root        *Configuration
		Development bool
	}
	// AppConfiguration is the application level configuration,include all of component's configurations
	AppConfiguration struct {
		*Configuration
	}
)

var defaultOptions = options{
	global: false,
}

// use the dir of app position as base dir, not pwd.
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
	New(WithGlobal(true))
}

// New create an application configuration instance.
//
// New as global by default, if you want to create a local configuration, set global to false.
// Initialization such as:
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

// NewFromStringMap create from a string map
func NewFromStringMap(data map[string]any, opts ...Option) *Configuration {
	p := NewParserFromStringMap(data)
	return NewFromParse(p, opts...)
}

// NewFromParse create from parser
func NewFromParse(parser *Parser, opts ...Option) *Configuration {
	cnf := New(opts...)
	// clear the local path
	cnf.opts.localPath = ""
	cnf.parser = parser
	return cnf
}

// Exists check if the default configuration file exists
func (c *Configuration) Exists() bool {
	_, err := os.Stat(c.opts.localPath)
	return err == nil
}

// LocalPath return the local configuration file path
func (c *Configuration) LocalPath() string {
	return c.opts.localPath
}

// AsGlobal set the Configuration as global
func (c *Configuration) AsGlobal() *Configuration {
	global.Configuration = c
	c.opts.global = true
	return c
}

// Load parse configuration from file. panic if load failed, it is better call it when start up your application.
func (c *Configuration) Load() *Configuration {
	if err := c.loadInternal(); err != nil {
		panic("config load error:" + err.Error())
	}
	c.Development = c.parser.k.Bool("development")
	return c
}

// Reload configuration,if only work for local path configuration
func (c *Configuration) Reload() {
	c.parser = nil
	c.Load()
}

// load configuration,if the RemoteProvider is set,will ignore local configuration
func (c *Configuration) loadInternal() (err error) {
	// if parser is nil, use default local config file
	if c.parser == nil {
		c.parser = NewParser()
	}
	if c.opts.localPath != "" {
		TryLoadEnvFromFile(filepath.Dir(c.opts.localPath), "")
		err = c.parser.LoadFileWithEnv(c.opts.localPath)
		if err != nil {
			return err
		}
	}
	const ifs = "includeFiles"
	if c.IsSet(ifs) {
		c.opts.includeFiles = c.StringSlice(ifs)
		for i, v := range c.opts.includeFiles {
			path, err := tryAbs(c.GetBaseDir(), v)
			if err != nil {
				panic(fmt.Errorf("config file no exists:%s,cause by:%s", path, err))
			}
			c.opts.includeFiles[i] = path
		}
	}
	// make sure the "include files" in attach files is not working
	copyifs := make([]string, len(c.opts.includeFiles))
	copy(copyifs, c.opts.includeFiles)
	for _, attach := range copyifs {
		if err = c.parser.LoadFileWithEnv(attach); err != nil {
			return err
		}
	}
	return err
}

// tryAbs return the absolute path by the base dir if path is a relative path.
func tryAbs(basedir, path string) (string, error) {
	if !filepath.IsAbs(path) {
		// first try to load from the base dir
		rpath := filepath.Join(basedir, path)
		_, err := os.Stat(rpath)
		if err == nil {
			return rpath, err
		}
		// if not found, try to load from the current working directory
		apath, err := filepath.Abs(path)
		if err != nil {
			return path, err
		}
		path = apath
	}
	_, err := os.Stat(path)
	return path, err
}

// Merge an input config stream, parameter b is YAML stream
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

// ParserOperator return the underlying parser that converts bytes to map
func (c *Configuration) ParserOperator() *koanf.Koanf {
	return c.parser.k
}

// Sub return a new Configuration by a sub node.return current node if path empty,panic if path not found.
//
// sub node keeps the same root configuration of the current node.
func (c *Configuration) Sub(path string) *Configuration {
	if path == "" {
		return c
	}
	p, err := c.Parser().Sub(path)
	if err != nil {
		panic(err)
	}
	nc := &Configuration{
		opts:        c.opts,
		parser:      p,
		Development: c.Development,
	}
	c.passRoot(nc)
	return nc
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

// Each iterates the slice path of the Configuration.Slice item can be a map or a slice.
// if the item is a map, the root value will be the first key in the map.
// if the item is a slice, the root value will be empty.
// slice example:
//
//	path:
//	- slice: "sv"
//	- slice: "sv2"
//	- slice: "sv3"
//
// map example:
//
//	path:
//	- key:
//	    sub: "sv"
//
// because koanf should be map value, so not support like:
//
//	path:
//	- string
//	- string2
func (c *Configuration) Each(path string, cb func(root string, sub *Configuration)) {
	ops := c.ParserOperator().Slices(path)
	// check sub item has root value
	var getRoot = func(kf *koanf.Koanf) string {
		raw := kf.Raw()
		for k, _ := range raw {
			if len(raw) == 1 {
				return k
			} else {
				break
			}

		}
		return ""
	}
	for _, op := range ops {
		if root := getRoot(op); root != "" {
			kf := op.Cut(root)
			if len(kf.Keys()) == 0 {
				cb(root, c.CutFromOperator(op))
			} else {
				cb(root, c.CutFromOperator(kf))
			}
		} else {
			cb(root, c.CutFromOperator(op))
		}
	}
}

// Map iterates the map path of the Configuration.It is sort by sort.Strings,
// no guarantee of the order in the configuration file.
// Each key in the path must be a map value, otherwise it will panic.
func (c *Configuration) Map(path string, cb func(root string, sub *Configuration)) {
	ops := c.ParserOperator().MapKeys(path)
	for _, op := range ops {
		cb(op, c.Sub(Join(path, op)))
	}
}

// Copy return a new copied Configuration
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
// returns the path 'appName' value if exists
func (c *Configuration) AppName() string {
	return c.String("appName")
}

// Version indicates the application's sem version
//
// returns the path 'version' value, no set return empty
func (c *Configuration) Version() string {
	return c.String("version")
}

// Unmarshal map data of config into a struct, values are merged.
//
// Tags on the fields of the structure must be properly set.
// Notice: if use nested struct, should tag `,inline` and can't use pointer: like
//
//		type Config struct {
//		    BaseConfig `yaml:",inline"`
//	     // should not *BaseConfig
//	 }
func (c *Configuration) Unmarshal(dst any) (err error) {
	return c.parser.Unmarshal("", dst)
}

// Abs returns the absolute path by the base dir if path is a relative path
func Abs(path string) string { return global.Abs(path) }

// Abs returns the absolute path by the base dir if path is a relative path,if path is empty return empty.
func (c *Configuration) Abs(path string) string {
	if path == "" {
		return ""
	}
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
	return c.parser.k.Raw()
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
	if len(global.Configuration.AllSettings()) == 0 {
		global.Configuration.Load()
	}
	return &global
}
