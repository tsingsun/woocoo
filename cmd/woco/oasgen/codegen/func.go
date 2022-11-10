package codegen

import (
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	Funcs = template.FuncMap{
		"lower":     strings.ToLower,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"upper":     strings.ToUpper,
		"trim":      strings.Trim,
		"replace":   strings.ReplaceAll,
		"hasField":  helper.HasField,
		"pascal":    helper.Pascal,
		"base":      filepath.Base,
		"extend":    extend,
		"pkgName":   code.PkgShortName,
	}
)

// graphScope wraps the Graph object with extended scope.
type graphScope struct {
	*Graph
	Scope map[any]any
}

// extend extends the parent block with a KV pairs.
//
//	{{ with $scope := extend $ "key" "value" }}
//		{{ template "setters" $scope }}
//	{{ end}}
func extend(v any, kv ...any) (any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("invalid number of parameters: %d", len(kv))
	}
	scope := make(map[any]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		scope[kv[i]] = kv[i+1]
	}
	switch v := v.(type) {
	case *Graph:
		return &graphScope{Graph: v, Scope: scope}, nil
	default:
		return nil, fmt.Errorf("invalid type for extend: %T", v)
	}
}

func schemaNameFromRef(ref string) string {
	ss := strings.Split(ref, "/")
	return helper.Pascal(ss[len(ss)-1])
}

func ModelMapToTypeInfo(model map[string]*ModelMap) (map[string]*code.TypeInfo, error) {
	m := make(map[string]*code.TypeInfo)
	for k, v := range model {
		pkg, id := code.PkgAndType(v.Model)
		pn := code.PkgShortName(pkg)
		m[k] = &code.TypeInfo{
			Ident:   pn + "." + id,
			PkgName: pn,
			PkgPath: pkg,
		}
	}
	return m, nil
}