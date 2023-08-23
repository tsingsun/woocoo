package oasgen

import (
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"github.com/tsingsun/woocoo/web/handler"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	pathParamRE *regexp.Regexp
	pattenMap   map[string]string
	title       = cases.Title(language.English, cases.NoLower)

	funcs = template.FuncMap{
		"extend":             extend,
		"oasUriToGinUri":     OasUriToGinUri,
		"ginReturnType":      GinReturnType,
		"patternMap":         func() map[string]string { return pattenMap },
		"isSupportNegotiate": IsSupportNegotiate,
		"isBytes":            IsBytes,
		"normalizePkg": func(pkg string) string {
			pkg, err := helper.NormalizePkg(pkg)
			if err != nil {
				panic(err)
			}
			return filepath.Base(pkg)
		},
		"hasTag": func(ts []string, t string) bool {
			return HasTag(ts, t)
		},
		"sortAsc": sortAsc,
	}
)

func init() {
	pathParamRE = regexp.MustCompile("{[.;?]?([^{}*]+)\\*?}")
	pattenMap = make(map[string]string)
}

func AddPattern(pattern string) (key string) {
	if v, ok := pattenMap[pattern]; ok {
		return v
	}
	l := len(pattenMap)
	key = fmt.Sprintf("oas_pattern_%d", l)
	pattenMap[pattern] = key
	return key
}

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
	if ref == "" {
		return ""
	}
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

// OasUriToGinUri converts a swagger style path URI with parameters to a
// Gin compatible path URI. We need to replace all Swagger parameters with
// ":param". Valid input parameters are:
//
//	{param}
//	{param*}
//	{.param}
//	{.param*}
//	{;param}
//	{;param*}
//	{?param}
//	{?param*}
func OasUriToGinUri(uri string) string {
	return pathParamRE.ReplaceAllString(uri, ":$1")
}

// GinReturnType returns the which gin call for the incoming content type
func GinReturnType(httpContent string) string {
	switch httpContent {
	case "application/json":
		return "JSON"
	case "application/xml":
		return "XML"
	case "text/plain":
		return "String"
	case "text/html":
		return "HTML"
	default:
		return "String"
	}
}

func HasTag(src []string, tag string) bool {
	for _, v := range src {
		if strings.HasPrefix(v, tag) {
			return true
		}
	}
	return false
}

// IsSupportNegotiate check if the response content type is support negotiate.
// if the response content type is not support negotiate, the response content type will be set to the first content type in the response use bytes.
func IsSupportNegotiate(ress []string) bool {
	for _, res := range ress {
		for _, c := range handler.DefaultNegotiateFormat {
			if c == res {
				return true
			}
		}
	}
	return false
}

// IsBytes check if the response content type is bytes.
func IsBytes(p code.Type) bool {
	return p == code.TypeBytes
}

func sortAsc(ts []string) []string {
	sort.Strings(ts)
	return ts
}
