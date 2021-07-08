package conf

import (
	"fmt"
	"os"
)

// configuration detail
// attachFiles: the files will merge into main configuration and override it.
type options struct {
	localPath   string
	attachFiles []string
	//use parser global
	global bool
}

// Option the function to apply configuration option
type Option func(*options)

// LocalPath init local instance file path
// A s is file path
func LocalPath(s string) Option {
	return func(o *options) {
		_, err := os.Stat(s)
		if err != nil {
			panic(fmt.Sprintf("local file '%s' is not exists", s))
		}
		o.localPath = s
	}
}

// AttachFiles 附加文件中的配置将会重写主配置文件,对于非法的文件,将被忽略.
// you can set a configuration for dev ENV,but attach instance only effect in local file configuration
func AttachFiles(paths ...string) Option {
	return func(o *options) {
		for _, s := range paths {
			_, err := os.Stat(s)
			if err != nil {
				panic(fmt.Errorf("attach config file %s error,%s", s, err))
			}
			o.attachFiles = append(o.attachFiles, s)
		}
	}
}

func Global(isGlobal bool) Option {
	return func(o *options) {
		o.global = isGlobal
	}
}
