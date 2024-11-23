package conf

import (
	"fmt"
	"os"
	"path/filepath"
)

// configuration detail
// includeFiles: the files will merge into main configuration and override it.
type options struct {
	localPath    string
	basedir      string
	includeFiles []string
	// use parser global
	global bool
}

// Option the function to apply configuration option
type Option func(*options)

// WithLocalPath use a local configuration file. if used, try local env files(named: env or env.local) in dir of the local file.
func WithLocalPath(path string) Option {
	return func(o *options) {
		abs, err := tryAbs(o.basedir, path)
		if err != nil {
			panic(fmt.Sprintf("local file %q is not exists", path))
		}
		o.localPath = abs
	}
}

// WithBaseDir init base directory where configuration files location, usually is the directory which application executable file is in
// parameter s can be an absolute path or relative path. Note that It also sets default config file name to "etc/app.yaml"
func WithBaseDir(path string) Option {
	return func(o *options) {
		var err error
		o.basedir, err = filepath.Abs(path)
		if err != nil {
			panic(fmt.Sprintf("base dir %q is not exists", path))
		}
		o.localPath = filepath.Join(o.basedir, defaultConfigFile)
	}
}

// WithIncludeFiles init include files
//
// The configuration in the attached file will overwrite the master configuration file and will be ignored for invalid files.
// you can set a configuration for dev ENV,but attach instance only effect in local file configuration
func WithIncludeFiles(paths ...string) Option {
	return func(o *options) {
		for _, s := range paths {
			_, err := os.Stat(s)
			if err != nil {
				panic(fmt.Errorf("attach config file %q error,%s", s, err))
			}
			o.includeFiles = append(o.includeFiles, s)
		}
	}
}

// WithGlobal indicate weather use as global configuration
func WithGlobal(g bool) Option {
	return func(o *options) {
		o.global = g
	}
}
