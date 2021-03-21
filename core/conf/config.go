package conf

import (
	"bytes"
	"errors"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"path/filepath"
	"time"
)

type Configurable interface {
	// initial from config
	Configurable(config *Config, key string)
}

type Config struct {
	opts  options
	viper *viper.Viper
}

var global *Config

// create an application configuration instance
func New() (*Config, error) {
	absPath, err := filepath.Abs(defaultOptions.localPath)
	if err != nil {
		return nil, err
	} else {
		defaultOptions.localPath = absPath
	}

	opts := defaultOptions
	cnf := &Config{
		opts:  opts,
		viper: viper.GetViper(),
	}
	return cnf, nil
}

// create an application configuration instance with options
func NewWithOption(opt ...Option) (*Config, error) {
	cnf, err := New()
	if err != nil {
		return nil, err
	}
	for _, o := range opt {
		o(&cnf.opts)
	}
	if !cnf.opts.global {
		cnf.viper = viper.New()
	}
	cnf.viper.SetConfigType("yaml")

	return cnf, nil
}

func (c Config) AsGlobal() {
	global = &c
}

func (c *Config) Load() *Config {
	if err := c.loadInternal(); err != nil {
		panic("instance error:" + err.Error())
	}
	return c
}

// load configuration,if the RemoteProvider is set,will ignore local configuration
func (c *Config) loadInternal() (err error) {
	opts := c.opts
	if opts.localPath != "" && opts.remoteProvider.Provider() != "" {
		return errors.New("local and remote are both config,but only remote configuration will be work")
	}
	vp := c.viper
	if opts.remoteProvider.Provider() != "" {
		if opts.remoteProvider.SecretKeyring() == "" {
			if err = vp.AddRemoteProvider(opts.remoteProvider.Provider(), opts.remoteProvider.endpoint, opts.remoteProvider.path); err != nil {
				return
			}
		} else {
			if err = vp.AddSecureRemoteProvider(opts.remoteProvider.Provider(), opts.remoteProvider.Endpoint(),
				opts.remoteProvider.Path(), opts.remoteProvider.SecretKeyring()); err != nil {
				return
			}
		}
		if err = vp.ReadRemoteConfig(); err != nil {
			return
		}
	} else {
		vp.SetConfigFile(c.opts.localPath)
		if err = vp.ReadInConfig(); err != nil {
			return
		}
		for _, attach := range opts.attachFiles {
			var bts []byte
			if bts, err = afero.ReadFile(afero.NewOsFs(), attach); err != nil {
				return
			} else if err = vp.MergeConfig(bytes.NewReader(bts)); err != nil {
				return
			}
		}

	}
	return err
}

func (c *Config) WatchConfig() error {
	var err error
	if c.opts.remoteProvider.Provider() != "" {
		err = c.viper.WatchRemoteConfigOnChannel()
	} else {
		c.viper.WatchConfig()
	}
	return err
}

func (c Config) IsDebug() bool {
	return c.opts.isDebug
}

func Get(key string) interface{} { return global.Get(key) }
func (c *Config) Get(key string) interface{} {
	return c.viper.Get(key)
}

func GetBool(key string) bool { return global.GetBool(key) }
func (c *Config) GetBool(key string) bool {
	return c.viper.GetBool(key)
}

func GetFloat64(key string) float64 { return global.GetFloat64(key) }
func (c *Config) GetFloat64(key string) float64 {
	return c.viper.GetFloat64(key)
}

func GetInt(key string) int { return global.GetInt(key) }
func (c *Config) GetInt(key string) int {
	return c.viper.GetInt(key)
}

func GetIntSlice(key string) []int { return global.GetIntSlice(key) }
func (c *Config) GetIntSlice(key string) []int {
	return c.viper.GetIntSlice(key)
}

func GetString(key string) string { return global.GetString(key) }
func (c *Config) GetString(key string) string {
	return c.viper.GetString(key)
}

func GetStringMap(key string) map[string]interface{} { return global.GetStringMap(key) }
func (c *Config) GetStringMap(key string) map[string]interface{} {
	return c.viper.GetStringMap(key)
}

func GetStringMapString(key string) map[string]string { return global.GetStringMapString(key) }
func (c *Config) GetStringMapString(key string) map[string]string {
	return c.viper.GetStringMapString(key)
}

func GetStringSlice(key string) []string { return global.GetStringSlice(key) }
func (c *Config) GetStringSlice(key string) []string {
	return c.viper.GetStringSlice(key)
}

func GetTime(key string) time.Time { return global.GetTime(key) }
func (c *Config) GetTime(key string) time.Time {
	return c.viper.GetTime(key)
}

func GetDuration(key string) time.Duration { return global.GetDuration(key) }
func (c *Config) GetDuration(key string) time.Duration {
	return c.viper.GetDuration(key)
}

func IsSet(key string) bool { return global.IsSet(key) }
func (c *Config) IsSet(key string) bool {
	return c.viper.IsSet(key)
}

func AllSettings() map[string]interface{} { return global.AllSettings() }
func (c *Config) AllSettings() map[string]interface{} {
	return c.viper.AllSettings()
}

func Operator() *viper.Viper { return global.Operator() }

// return configuration operator
func (c Config) Operator() *viper.Viper {
	return c.viper
}
