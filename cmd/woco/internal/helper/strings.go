package helper

import (
	"fmt"
	"github.com/tsingsun/woocoo/cmd/woco/gen"
	"go/token"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func StringToGoCommentWithPrefix(in, prefix string) string {
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

// PathMatch reports if the given name matches the extended pattern.
func PathMatch(patterns []string, name string) bool {
	for _, pat := range patterns {
		matched, _ := filepath.Match(pat, name)
		if matched {
			return true
		}
	}
	return false
}
