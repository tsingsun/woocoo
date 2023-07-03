package gen

import (
	"fmt"
	"golang.org/x/tools/imports"
	"os"
	"os/exec"
	"path/filepath"
)

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
			return fmt.Errorf("write file %q: %w", path, err)
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
		if err := FormatGoFile(path, content); err != nil {
			return err
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

func RunCmd(root string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run cmd %q %v in %q failed: %w", name, arg, root, err)
	}
	return nil
}

func FormatGoFile(path string, content []byte) error {
	src, err := imports.Process(path, content, nil)
	if err != nil {
		return fmt.Errorf("format file %s: %w", path, err)
	}
	if err := os.WriteFile(path, src, 0644); err != nil {
		return fmt.Errorf("format file:write file %s: %w", path, err)
	}
	return nil
}
