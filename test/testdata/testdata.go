package testdata

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"path/filepath"
	"runtime"
)

// basedir is the root directory of this package.
var (
	basedir           string
	DefaultConfigFile = "etc/app.yaml"
	Config            *conf.Configuration
)

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basedir = filepath.Dir(currentFile)
	Config = conf.New(conf.LocalPath(Path(DefaultConfigFile)), conf.BaseDir(basedir)).Load()
}

func BaseDir() string {
	return basedir
}

// Path returns the absolute path the given relative file or directory path,
// relative to the google.golang.org/hello_grpc/testdata directory in the user's GOPATH.
// If rel is already absolute, it is returned unmodified.
func Path(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}

	return filepath.Join(basedir, rel)
}

func Tmp(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}

	tmpPath := filepath.Join(filepath.Dir(basedir), "tmp")
	return filepath.Join(tmpPath, rel)
}
