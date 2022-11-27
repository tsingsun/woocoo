package helper

import (
	"fmt"
	"golang.org/x/tools/imports"
	"os"
	"os/exec"
	"path/filepath"
)

// Expect panics if the condition is false.
func Expect(cond bool, msg string, args ...any) {
	if !cond {
		panic(GraphError{fmt.Sprintf(msg, args...)})
	}
}

func CheckGraphError(err error, msg string, args ...any) {
	if err != nil {
		args = append(args, err)
		panic(GraphError{fmt.Sprintf(msg+": %s", args...)})
	}
}

type GraphError struct {
	msg string
}

func (p GraphError) Error() string { return fmt.Sprintf("entc/gen: %s", p.msg) }

func CatchGraphError(err *error) {
	if e := recover(); e != nil {
		gerr, ok := e.(GraphError)
		if !ok {
			panic(e)
		}
		*err = gerr
	}
}

type Assets struct {
	dirs  map[string]struct{}
	files map[string][]byte
}

func (a *Assets) Add(path string, content []byte) {
	if a.files == nil {
		a.files = make(map[string][]byte)
	}
	a.files[path] = content
}

func (a *Assets) AddDir(path string) {
	if a.dirs == nil {
		a.dirs = make(map[string]struct{})
	}
	a.dirs[path] = struct{}{}
}

// Write files and dirs in the Assets.
func (a Assets) Write() error {
	for dir := range a.dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("create dir %q: %w", dir, err)
		}
	}
	for path, content := range a.files {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("Write file %q: %w", path, err)
		}
	}
	return nil
}

// Format runs "goimports" on all Assets.
func (a Assets) Format() error {
	for path, content := range a.files {
		if filepath.Ext(path) != ".go" {
			continue
		}
		src, err := imports.Process(path, content, nil)
		if err != nil {
			return fmt.Errorf("Format file %s: %w", path, err)
		}
		if err := os.WriteFile(path, src, 0644); err != nil {
			return fmt.Errorf("Write file %s: %w", path, err)
		}
	}
	return nil
}

func (a Assets) ModTidy(root string) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = root
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stdout
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}
