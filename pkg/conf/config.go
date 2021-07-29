package conf

import (
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
	// Apply initial from config
	Apply(cfg *Configuration, path string)
}

type Configuration struct {
	opts        options
	parser      *Parser
	Development bool
}

var (
	global *Configuration
)

var defaultOptions = options{
	localPath: filepath.Join(filepath.Dir(os.Args[0]), "config/app.yaml"),
	basedir:   filepath.Dir(os.Args[0]),
	global:    true,
}

func init() {
	global = New()
}

// New create an application configuration instance
func New() *Configuration {
	opts := defaultOptions
	cnf := &Configuration{
		opts:   opts,
		parser: NewParser(),
	}
	return cnf
}

// BuildWithOption create an application configuration instance with options
func BuildWithOption(opt ...Option) (*Configuration, error) {
	cnf := New()
	for _, o := range opt {
		o(&cnf.opts)
	}
	if !cnf.opts.global {
		cnf.parser = NewParser()
	}
	if cnf.opts.global {
		cnf.asGlobal()
	}
	cnf.Build()
	return cnf, nil
}

func (c Configuration) asGlobal() {
	global = &c
}

func (c *Configuration) Build() *Configuration {
	if err := c.loadInternal(); err != nil {
		panic("instance error:" + err.Error())
	}
	c.Development = c.parser.k.Bool("development")
	return c
}

// load configuration,if the RemoteProvider is set,will ignore local configuration
func (c *Configuration) loadInternal() error {
	opts := c.opts

	absPath, err := filepath.Abs(opts.localPath)
	if err != nil {
		return err
	}

	c.parser, err = NewParserFromFile(absPath)
	if err != nil {
		return err
	}

	if c.IsSet("includeFiles") {
		for _, v := range c.StringSlice("includeFiles") {
			if filepath.IsAbs(v) {
				if _, err := os.Stat(v); err != nil {
					panic(fmt.Errorf("config file no exists:%s,cause by:%s", v, err))
				} else {
					opts.includeFiles = append(opts.includeFiles, v)
				}
			} else {
				path := filepath.Join(c.opts.basedir, v)
				if _, err := os.Stat(path); err != nil {
					panic(fmt.Errorf("config file no exists:%s,cause by:%s", path, err))
				} else {
					opts.includeFiles = append(opts.includeFiles, path)
				}
			}
		}
	}
	for _, attach := range opts.includeFiles {
		if err = c.parser.k.Load(file.Provider(attach), yaml.Parser()); err != nil {
			return err
		}
	}
	return err
}

func (c *Configuration) GetBaseDir() string {
	return c.opts.basedir
}

func (c Configuration) Abs(path string) string {
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

func IsSet(path string) bool { return global.IsSet(path) }
func (c *Configuration) IsSet(path string) bool {
	return c.parser.IsSet(path)
}

func AllSettings() map[string]interface{} { return global.AllSettings() }
func (c *Configuration) AllSettings() map[string]interface{} {
	return c.parser.k.All()
}

func Slices(path string) []*koanf.Koanf { return global.Slices(path) }
func (c *Configuration) Slices(path string) []*koanf.Koanf {
	return c.parser.k.Slices(path)
}

func Operator() *Parser { return global.Parser() }

// Parser return configuration operator
func (c Configuration) Parser() *Parser {
	return c.parser
}

func (c Configuration) ParserOperator() *koanf.Koanf {
	return c.parser.k
}

func (c Configuration) Sub(path string) *Configuration {
	p, err := c.Parser().Sub(path)
	if err != nil {
		return nil
	}
	return &Configuration{
		opts:        c.opts,
		parser:      p,
		Development: c.Development,
	}
}

func Join(ps ...string) string {
	return strings.Join(ps, KeyDelimiter)
}
