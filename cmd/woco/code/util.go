package code

import (
	"go/build"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var (
	title = cases.Title(language.English, cases.NoLower)
)

// PkgAndType take a string in the form github.com/package/blah.Type and split it into package and type
func PkgAndType(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", name
	}

	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

// NormalizeVendor takes a qualified package path and turns it into normal one.
// eg .
// github.com/foo/vendor/github.com/99designs/gqlgen/graphql becomes
// github.com/99designs/gqlgen/graphql
func NormalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}

// QualifyPackagePath takes an import and fully qualifies it with a vendor dir, if one is required.
// eg .
// github.com/99designs/gqlgen/graphql becomes
// github.com/foo/vendor/github.com/99designs/gqlgen/graphql
//
// x/tools/packages only supports 'qualified package paths' so this will need to be done prior to calling it
// See https://github.com/golang/go/issues/30289
func QualifyPackagePath(importPath string) string {
	wd, _ := os.Getwd()

	// in go module mode, the import path doesn't need fixing
	if _, ok := goModuleRoot(wd); ok {
		return importPath
	}

	pkg, err := build.Import(importPath, wd, 0)
	if err != nil {
		return importPath
	}

	return pkg.ImportPath
}

var invalidPackageNameChar = regexp.MustCompile(`[^\w]`)

func SanitizePackageName(pkg string) string {
	return invalidPackageNameChar.ReplaceAllLiteralString(filepath.Base(pkg), "_")
}

// Indirect returns the type at the end of indirection.
func Indirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func pkgPath(t reflect.Type) string {
	pkg := t.PkgPath()
	if pkg != "" {
		return pkg
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Ptr, reflect.Map:
		return pkgPath(t.Elem())
	}
	return pkg
}

// PkgName returns the package name from a Go
// identifier with a package qualifier.
func PkgName(ident string) string {
	i := strings.LastIndexByte(ident, '.')
	if i == -1 {
		return ""
	}
	s := ident[:i]
	if i := strings.LastIndexAny(s, "]*"); i != -1 {
		s = s[i+1:]
	}
	return s
}

// TypeName returns the element type name from a Go
// identifier with a package qualifier.
func TypeName(ident string) string {
	i := strings.LastIndexAny(ident, "]*")
	if i == -1 {
		if i := strings.LastIndexByte(ident, '.'); i != -1 {
			return ident[i+1:]
		}
		return ident
	}
	return ident[i+1:]
}

func PkgShortName(pkgPath string) string {
	parts := strings.Split(pkgPath, "/")
	return parts[len(parts)-1]
}
