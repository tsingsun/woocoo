package conf

import (
	"bytes"
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
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
	// cfg is the Configuration of the component, and it's sub configuration of root
	// path is the relative path to root,if root is the component,path will be empty
	// Apply also use for lazy load scene
	// notice: if any error in apply process,use panic to expose error,and stop application.
	Apply(cfg *Configuration)
}

const (
	baseDirEnv = "WOOCOO_BASEDIR"
)

var (
	global            AppConfiguration
	defaultConfigFile = "etc/app.yaml"
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
}

// New create an application configuration instance
func New(opts ...Option) *Configuration {
	cnf := &Configuration{
		opts: defaultOptions,
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
	if c.parser == nil {
		if c.opts.localPath == "" {
			LocalPath(defaultConfigFile)(&c.opts)
		}
		c.parser, err = NewParserFromFile(c.opts.localPath)
		if err != nil {
			return err
		}
	}

	if c.IsSet("includeFiles") {
		for _, v := range c.StringSlice("includeFiles") {
			if filepath.IsAbs(v) {
				if _, err := os.Stat(v); err != nil {
					panic(fmt.Errorf("config file no exists:%s,cause by:%s", v, err))
				} else {
					c.opts.includeFiles = append(c.opts.includeFiles, v)
				}
			} else {
				path := filepath.Join(c.opts.basedir, v)
				if _, err := os.Stat(path); err != nil {
					panic(fmt.Errorf("config file no exists:%s,cause by:%s", path, err))
				} else {
					c.opts.includeFiles = append(c.opts.includeFiles, path)
				}
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

// Global return default(global) Configuration instance
func Global() *AppConfiguration {
	if global.Configuration == nil || global.parser == nil {
		global.Configuration = New().Load()
		// panic("global configuration has not initialed.")
	}
	return &global
}

// Parser return configuration operator
func (c *Configuration) Parser() *Parser {
	return c.parser
}

func (c *Configuration) ParserFromBytes(bs []byte) error {
	p, err := NewParserFromBuffer(bytes.NewReader(bs))
	if err != nil {
		return err
	}
	c.parser = p
	return nil
}

func (c *Configuration) ParserOperator() *koanf.Koanf {
	return c.parser.k
}

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

func (c *Configuration) CutFromParser(p *Parser) *Configuration {
	nf := *c
	nf.parser = p
	c.passRoot(&nf)
	return &nf
}

func (c *Configuration) CutFromOperator(kf *koanf.Koanf) *Configuration {
	nf := *c
	nf.parser = &Parser{
		k: kf,
	}
	c.passRoot(&nf)
	return &nf
}

func (c Configuration) passRoot(sub *Configuration) {
	if c.root != nil {
		sub.root = c.root
	} else {
		sub.root = &c
	}
}

// SubOperator return the slice operator of the Configuration which is only contains slice path
func (c *Configuration) SubOperator(path string) (out []*koanf.Koanf, err error) {
	raw := c.parser.k.Raw()
	for k, v := range raw {
		if path != "" && k != path {
			continue
		}
		v, ok := v.([]interface{})
		if !ok {
			continue
		}
		for _, s := range v {
			mp, ok := s.(map[string]interface{})
			if !ok {
				continue
			}

			kf := koanf.New(KeyDelimiter)
			if err = kf.Load(confmap.Provider(mp, ""), nil); err != nil {
				return
			}
			out = append(out, kf)
		}
	}
	return
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

// Unmarshal map data of config into a struct.
//
// Tags on the fields of the structure must be properly set.
func (c *Configuration) Unmarshal(dst interface{}) (err error) {
	return c.parser.Unmarshal("", dst)
}

func (c *Configuration) IsSlice(path string) bool {
	if !c.IsSet(path) {
		return false
	}
	v := c.Get(path)
	_, ok := v.([]interface{})
	return ok
}

// Abs 返回绝对路径
func Abs(path string) string { return global.Abs(path) }
func (c *Configuration) Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.GetBaseDir(), path)
}

func Get(path string) interface{} { return global.Get(path) }
func (c *Configuration) Get(key string) interface{} {
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
func AllSettings() map[string]interface{} { return global.AllSettings() }
func (c *Configuration) AllSettings() map[string]interface{} {
	return c.parser.k.All()
}

// Join paths
func Join(ps ...string) string {
	return strings.Join(ps, KeyDelimiter)
}
