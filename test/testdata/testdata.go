package testdata

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// basedir is the root directory of this package.
var (
	basedir           string
	DefaultConfigFile = "etc/app.yaml"
)

func init() {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic(errors.New("runtime.Caller error at test init"))
	}
	basedir = filepath.Dir(currentFile)
}

func TestConfigFile() string {
	return Path(DefaultConfigFile)
}

func TestStruct() any {
	type test struct {
		String string
		Bool   bool
		Int    int
		Float  float64
		Array  []string
	}
	return &test{
		String: "string",
		Bool:   true,
		Int:    1,
		Float:  1.1,
		Array:  []string{"a", "b"},
	}
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
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		panic(err)
	}
	return filepath.Join(tmpPath, rel)
}

func FileBytes(rel string) []byte {
	f, err := os.Open(Path(rel))
	if err != nil {
		panic(err)
	}
	v, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return v
}
