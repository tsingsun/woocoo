package gen

import (
	"entgo.io/ent/entc/gen"
	"github.com/go-openapi/inflect"
	"github.com/tsingsun/woocoo/cmd/woco/code"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

var (
	Funcs = template.FuncMap{
		"lower":     strings.ToLower,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"upper":     strings.ToUpper,
		"trim":      strings.Trim,
		"replace":   strings.ReplaceAll,
		"hasField":  hasField,
		"pascal":    pascal,
		"base":      filepath.Base,
		"pkgName":   code.PkgShortName,
		"join":      join,
		"quote":     quote,
		"joinQuote": joinQuote,
		"snake":     gen.Funcs["snake"],
	}
	rules    = ruleset()
	acronyms = make(map[string]struct{})
)

// AddAcronym adds initialism to the global ruleset.
func AddAcronym(word string) {
	acronyms[word] = struct{}{}
	rules.AddAcronym(word)
}

func ruleset() *inflect.Ruleset {
	rules := inflect.NewDefaultRuleset()
	// Add common initialism from golint and more.
	for _, w := range []string{
		"ACL", "API", "ASCII", "AWS", "CPU", "CSS", "DNS", "EOF", "GB", "GUID",
		"HCL", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "KB", "LHS", "MAC",
		"MB", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SQL", "SSH", "SSO",
		"TCP", "TLS", "TTL", "UDP", "UI", "UID", "URI", "URL", "UTF8", "UUID",
		"VM", "XML", "XMPP", "XSRF", "XSS",
	} {
		acronyms[w] = struct{}{}
		rules.AddAcronym(w)
	}
	return rules
}

// hasField determines if a struct has a field with the given name.
func hasField(v any, name string) bool {
	vr := reflect.Indirect(reflect.ValueOf(v))
	return vr.FieldByName(name).IsValid()
}

func isSeparator(r rune) bool {
	return r == '_' || r == '-' || unicode.IsSpace(r)
}

func pascalWords(words []string) string {
	for i, w := range words {
		upper := strings.ToUpper(w)
		if _, ok := acronyms[upper]; ok {
			words[i] = upper
		} else {
			words[i] = rules.Capitalize(w)
		}
	}
	return strings.Join(words, "")
}

// pascal converts the given name into a PascalCase.
//
//	user_info 	=> UserInfo
//	full_name 	=> FullName
//	user_id   	=> UserID
//	full-admin	=> FullAdmin
func pascal(s string) string {
	words := strings.FieldsFunc(s, isSeparator)
	return pascalWords(words)
}

// join is a wrapper around strings.Join to provide consistent output.
func join(a []string, sep string) string {
	sort.Strings(a)
	return strings.Join(a, sep)
}

// quote only strings.
func quote(v any) any {
	if s, ok := v.(string); ok {
		return strconv.Quote(s)
	}
	return v
}

func joinQuote(a []string, sep string) string {
	sort.Strings(a)
	for i, s := range a {
		a[i] = strconv.Quote(s)
	}
	return strings.Join(a, sep)
}
