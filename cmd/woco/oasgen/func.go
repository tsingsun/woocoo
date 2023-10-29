package oasgen

import (
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"github.com/tsingsun/woocoo/cmd/woco/internal/helper"
	"github.com/tsingsun/woocoo/web/handler"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	pathParamRE *regexp.Regexp
	pattenMap   map[string]string

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
		"hasTag":  HasTag,
		"sortAsc": sortAsc,
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
		"toStringFunc":                toStringFunc,
		"hasDefault":                  hasDefault,
		"printSchemaDefault":          printSchemaDefault,
		"canIgnorePointer":            canIgnorePointer,
		"stringToGoCommentWithPrefix": helper.StringToGoCommentWithPrefix,
	}
)

func init() {
	pathParamRE = regexp.MustCompile("{[.;?]?([^{}*]+)\\*?}")
	pattenMap = make(map[string]string)
}

// AddPattern add a Openapi3.schema.Pattern to the pattern map, and return the key of the pattern.
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

// HasTag check if the src has an element which start with tag.
func HasTag(src []string, tag string) bool {
	for _, v := range src {
		if strings.HasPrefix(v, tag) {
			return true
		}
	}
	return false
}

func lowCamelFirst(s string) string {
	sk := strings.Split(helper.Snake(s), "_")
	return sk[0] + s[len(sk[0]):]
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

func canIgnorePointer(sch *Schema) bool {
	typ := sch.Type.Type
	if typ.Numeric() || typ.Float() || typ.Integer() {
		return true
	}
	switch typ {
	case code.TypeString, code.TypeBytes, code.TypeBool, code.TypeTime, code.TypeJSON:
		return true
	}
	if sch.ItemSchema != nil {
		return true
	}
	if sch.Type.Ident != "" {
		if strings.HasPrefix(sch.Type.Ident, "[]") ||
			strings.HasPrefix(sch.Type.Ident, "map[") {
			return true
		}
	}
	return false
}

func toStringFunc(sch *Schema, name string, body bool) string {
	if sch.Type.Stringer() {
		return fmt.Sprintf("%s.String()", name)
	}
	typ := sch.Type

	if !body && typ.Nillable {
		name = "*" + name
	}
	switch t := typ.Type; {
	case t == code.TypeString:
		return name
	case t >= code.TypeInt8 && t <= code.TypeUint64:
		return fmt.Sprintf("strconv.FormatInt(int64(%s), 10)", name)
	case t >= code.TypeFloat32 && t <= code.TypeFloat64:
		return fmt.Sprintf("strconv.FormatFloat(float64(%s), 'f', -1, 64)", name)
	case t == code.TypeBool:
		return fmt.Sprintf("strconv.FormatBool(%s)", name)
	case t == code.TypeBytes:
		return fmt.Sprintf("string(%s)", name)
	default:
		//return fmt.Sprintf(`toStringFunc(%s,"%s",%v)`, name, para.Spec.Style, para.Schema.IsArray)
	}
	return fmt.Sprintf(`fmt.Sprintf("%%v",%s)`, name)
}

func printSchemaDefault(sch Schema) string {
	dv := sch.Spec.Value.Default
	if dv == nil {
		return ""
	}
	typ := sch.Type.Type
	switch typ {
	case code.TypeString:
		if sch.Type.Nillable {
			return fmt.Sprintf(`gds.Ptr("%s")`, dv)
		}
		return fmt.Sprintf(`"%s"`, dv)
	case code.TypeBool:
		if sch.Type.Nillable {
			return fmt.Sprintf(`gds.Ptr(%v)`, dv)
		}
		return fmt.Sprintf(`%v`, dv)
	case code.TypeInt, code.TypeInt8, code.TypeInt16, code.TypeInt32, code.TypeInt64,
		code.TypeUint, code.TypeUint8, code.TypeUint16, code.TypeUint32, code.TypeUint64:
		switch v := sch.Spec.Value.Default.(type) {
		case int:
			if sch.Type.Nillable {
				return fmt.Sprintf(`gds.Ptr(%s(%d))`, sch.StructString(), v)
			}
			return fmt.Sprintf(`%d`, v)
		case float64:
			if sch.Type.Nillable {
				return fmt.Sprintf(`gds.Ptr(%s(%d))`, sch.StructString(), int(v))
			}
			return fmt.Sprintf(`%d`, int(v))
		}
	case code.TypeFloat32, code.TypeFloat64:
		if sch.Type.Nillable {
			return fmt.Sprintf(`gds.Ptr(%s(%f))`, sch.StructString(), dv)
		}
		return fmt.Sprintf(`%v`, dv)
	case code.TypeTime:
		if sch.Type.Nillable {
			return fmt.Sprintf(`gds.Ptr(time.Parse(time.RFC3339,%s))`, dv)
		}
		return fmt.Sprintf(`time.Parse(time.RFC3339,%s)`, dv)
	}
	return fmt.Sprintf(`"%v"`, dv)
}

func hasDefault(ps []*Parameter) bool {
	for _, p := range ps {
		if p.Schema.Spec.Value.Default != nil {
			return canPrintDefault(p.Schema)
		}
	}
	return false
}

func canPrintDefault(sch *Schema) bool {
	if sch.Spec.Value.Default == nil {
		return false
	}
	switch sch.Type.Type {
	case code.TypeString:
		return true
	case code.TypeBool:
		return true
	case code.TypeInt, code.TypeInt8, code.TypeInt16, code.TypeInt32, code.TypeInt64,
		code.TypeUint, code.TypeUint8, code.TypeUint16, code.TypeUint32, code.TypeUint64:
		return true
	case code.TypeFloat32, code.TypeFloat64:
		return true
	case code.TypeTime:
		return true
	}

	return false
}
