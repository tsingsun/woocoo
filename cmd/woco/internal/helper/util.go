package helper

import (
	"bufio"
	"embed"
	"entgo.io/ent/entc/gen"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func ParseT(path string, templates embed.FS, funcs template.FuncMap) *gen.Template {
	return gen.MustParse(gen.NewTemplate(path).
		Funcs(gen.Funcs).
		Funcs(funcs).
		ParseFS(templates, path))
}

// IsBuildError reports if the given error is an error from the Go command (e.g. syntax error).
func IsBuildError(err error) bool {
	if strings.HasPrefix(err.Error(), "entc/load: #") {
		return true
	}
	for _, s := range []string{
		"syntax error",
		"previous declaration",
		"invalid character",
		"could not import",
		"found '<<'",
	} {
		if strings.Contains(err.Error(), s) {
			return true
		}
	}
	return false
}

// CheckDir checks the given dir and reports if there are any VCS conflicts.
func CheckDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && dir != path {
			return filepath.SkipDir
		}
		return checkFile(path)
	})
}

// conflictMarker holds the default marker string for
// both Git and Mercurial (default length is 7).
const conflictMarker = "<<<<<<<"

// checkFile checks the given file line by line
// and reports if it contains any VCS conflicts.
func checkFile(path string) error {
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fi.Close()

	scan := bufio.NewScanner(fi)
	scan.Split(bufio.ScanLines)
	for i := 0; scan.Scan(); i++ {
		if l := scan.Text(); strings.HasPrefix(l, conflictMarker) {
			return fmt.Errorf("vcs conflict %s:%d", path, i+1)
		}
	}
	return nil
}
