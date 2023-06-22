package helper

import (
	"entgo.io/ent/entc/gen"
	"fmt"
	"go/token"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Camel converts the given name into a camelCase.
//
//	user_info  => userInfo
//	full_name  => fullName
//	user_id    => userID
//	full-admin => fullAdmin
func Camel(s string) string {
	cf := gen.Funcs["camel"].(func(string) string)
	return cf(s)
}

func stringToGoCommentWithPrefix(in, prefix string) string {
	if len(in) == 0 || len(strings.TrimSpace(in)) == 0 { // ignore empty comment
		return ""
	}

	// Normalize newlines from Windows/Mac to Linux
	in = strings.Replace(in, "\r\n", "\n", -1)
	in = strings.Replace(in, "\r", "\n", -1)

	// Add comment to each line
	var lines []string
	for i, line := range strings.Split(in, "\n") {
		s := "//"
		if i == 0 && len(prefix) > 0 {
			s += " " + prefix
		}
		lines = append(lines, fmt.Sprintf("%s %s", s, line))
	}
	in = strings.Join(lines, "\n")

	// in case we have a multiline string which ends with \n, we would generate
	// empty-line-comments, like `// `. Therefore remove this line comment.
	in = strings.TrimSuffix(in, "\n// ")
	return in
}

// EscapePathElements breaks apart a path, and looks at each element. If it's
// not a path parameter, eg, {param}, it will URL-escape the element.
func EscapePathElements(path string) string {
	elems := strings.Split(path, "/")
	for i, e := range elems {
		if strings.HasPrefix(e, "{") && strings.HasSuffix(e, "}") {
			// This is a path parameter, we don't want to mess with its value
			continue
		}
		elems[i] = url.QueryEscape(e)
	}
	return strings.Join(elems, "/")
}

func InStrSlice(haystack []string, needle string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}

	return false
}

// Snake converts the given struct or field name into a snake_case.
//
//	Username => username
//	FullName => full_name
//	HTTPCode => http_code
func Snake(s string) string {
	cf := gen.Funcs["snake"].(func(string) string)
	return cf(s)
}

// HasField determines if a struct has a field with the given name.
func HasField(v any, name string) bool {
	cf := gen.Funcs["hasField"].(func(any, string) bool)
	return cf(v, name)
}

// Pascal converts the given name into a PascalCase.
//
//	user_info 	=> UserInfo
//	full_name 	=> FullName
//	user_id   	=> UserID
//	full-admin	=> FullAdmin
func Pascal(s string) string {
	cf := gen.Funcs["pascal"].(func(string) string)
	return cf(s)
}

// Quote only strings.
func Quote(v any) any {
	cf := gen.Funcs["quote"].(func(any) any)
	return cf(v)
}

// Join is a wrapper around strings.Join to provide consistent output.
func Join(a []string, sep string) string {
	cf := gen.Funcs["join"].(func([]string, string) string)
	return cf(a, sep)
}

func JoinQuote(a []string, sep string) string {
	sort.Strings(a)
	for i, s := range a {
		a[i] = strconv.Quote(s)
	}
	return strings.Join(a, sep)
}

func NormalizePkg(pkg string) (nlpkg string, err error) {
	nlpkg = pkg
	base := path.Base(pkg)
	if strings.ContainsRune(base, '-') {
		base = strings.ReplaceAll(base, "-", "_")
		nlpkg = path.Join(path.Dir(nlpkg), base)
	}
	if !token.IsIdentifier(base) {
		err = fmt.Errorf("invalid package identifier: %q", base)
		return
	}
	return
}
