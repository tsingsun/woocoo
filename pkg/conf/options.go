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
	//use parser global
	global bool
}

// Option the function to apply configuration option
type Option func(*options)

// LocalPath init local instance file path
// A s is file path
func LocalPath(s string) Option {
	return func(o *options) {
		if !filepath.IsAbs(s) {
			s = filepath.Join(o.basedir, s)
		}
		_, err := os.Stat(s)
		if err != nil {
			panic(fmt.Sprintf("local file '%s' is not exists", s))
		}
		o.localPath = s
	}
}

func BaseDir(s string) Option {
	return func(o *options) {
		o.basedir = s
	}
}

// IncludeFiles 附加文件中的配置将会重写主配置文件,对于非法的文件,将被忽略.
// you can set a configuration for dev ENV,but attach instance only effect in local file configuration
func IncludeFiles(paths ...string) Option {
	return func(o *options) {
		for _, s := range paths {
			_, err := os.Stat(s)
			if err != nil {
				panic(fmt.Errorf("attach config file %s error,%s", s, err))
			}
			o.includeFiles = append(o.includeFiles, s)
		}
	}
}
