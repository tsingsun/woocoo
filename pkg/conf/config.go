package conf

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"os"
	"path/filepath"
	"time"
)

// Configurable can initial by framework
type Configurable interface {
	// Apply initial from config
	Apply(cnf *Config, path string)
}

type Config struct {
	opts    options
	parser  *Parser
	basedir string
}

var (
	global  *Config
	BaseDir = filepath.Dir(os.Args[0])
)

var defaultOptions = options{
	localPath: BaseDir + "./app.yaml",
	global:    true,
}

func init() {
	global = New()
}

// New create an application configuration instance
func New() *Config {
	opts := defaultOptions
	cnf := &Config{
		opts:   opts,
		parser: NewParser(),
	}
	return cnf
}

// BuildWithOption create an application configuration instance with options
func BuildWithOption(opt ...Option) (*Config, error) {
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

func (c Config) asGlobal() {
	global = &c
}

func (c *Config) Build() *Config {
	if err := c.loadInternal(); err != nil {
		panic("instance error:" + err.Error())
	}
	return c
}

// load configuration,if the RemoteProvider is set,will ignore local configuration
func (c *Config) loadInternal() error {
	opts := c.opts

	absPath, err := filepath.Abs(opts.localPath)
	if err != nil {
		return err
	}

	c.parser, err = NewParserFromFile(absPath)
	if err != nil {
		return err
	}

	for _, attach := range opts.attachFiles {
		if err = c.parser.k.Load(file.Provider(attach), yaml.Parser()); err != nil {
			return err
		}
	}
	return err
}

func Get(path string) interface{} { return global.Get(path) }
func (c *Config) Get(key string) interface{} {
	return c.parser.Get(key)
}

func Bool(path string) bool { return global.Bool(path) }
func (c *Config) Bool(path string) bool {
	return c.parser.k.Bool(path)
}

func Float64(path string) float64 { return global.Float64(path) }
func (c *Config) Float64(path string) float64 {
	return c.parser.k.Float64(path)
}

func Int(path string) int { return global.Int(path) }
func (c *Config) Int(path string) int {
	return c.parser.k.Int(path)
}

func IntSlice(path string) []int { return global.IntSlice(path) }
func (c *Config) IntSlice(path string) []int {
	return c.parser.k.Ints(path)
}

func String(path string) string { return global.String(path) }
func (c *Config) String(path string) string {
	return c.parser.k.String(path)
}

func StringMap(path string) map[string]string { return global.StringMap(path) }
func (c *Config) StringMap(path string) map[string]string {
	return c.parser.k.StringMap(path)
}

func StringSlice(path string) []string { return global.StringSlice(path) }
func (c *Config) StringSlice(path string) []string {
	return c.parser.k.Strings(path)
}

func Time(path string, layout string) time.Time { return global.Time(path, layout) }
func (c *Config) Time(path string, layout string) time.Time {
	return c.parser.k.Time(path, layout)
}

func Duration(path string) time.Duration { return global.Duration(path) }
func (c *Config) Duration(path string) time.Duration {
	return c.parser.k.Duration(path)
}

func IsSet(path string) bool { return global.IsSet(path) }
func (c *Config) IsSet(path string) bool {
	return c.parser.IsSet(path)
}

func AllSettings() map[string]interface{} { return global.AllSettings() }
func (c *Config) AllSettings() map[string]interface{} {
	return c.parser.k.All()
}

func Operator() *Parser { return global.Operator() }

// Operator return configuration operator
func (c Config) Operator() *Parser {
	return c.parser
}
