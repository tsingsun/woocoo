package helper

import (
	"fmt"
	"github.com/go-openapi/inflect"
	"go/token"
	"net/url"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	rules    = ruleset()
	acronyms = make(map[string]struct{})
)

func ruleset() *inflect.Ruleset {
	rules := inflect.NewDefaultRuleset()
	// Add common initialism from golint and more.
	for _, w := range []string{
		"ACL", "API", "ASCII", "AWS", "CPU", "CSS", "DNS", "EOF", "GB", "GUID",
		"HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "KB", "LHS", "MAC", "MB",
		"QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SQL", "SSH", "SSO", "TCP",
		"TLS", "TTL", "UDP", "UI", "UID", "URI", "URL", "UTF8", "UUID", "VM",
		"XML", "XMPP", "XSRF", "XSS",
	} {
		acronyms[w] = struct{}{}
		rules.AddAcronym(w)
	}
	return rules
}

// AddAcronym adds initialism to the global ruleset.
func AddAcronym(word string) {
	acronyms[word] = struct{}{}
	rules.AddAcronym(word)
}

func CreateDir(target string) error {
	_, err := os.Stat(target)
	if err == nil || !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		return fmt.Errorf("creating schema directory: %w", err)
	}
	return nil
}

var (
	separatorSet map[rune]struct{}
)

func init() {
	separatorSet = make(map[rune]struct{})
	for _, v := range "-#@!$&=.+:;_~ (){}[]" {
		separatorSet[v] = struct{}{}
	}
}

// CamelCaseEx will convert query-arg style strings to CamelCase. We will
// use `., -, +, :, ;, _, ~, ' ', (, ), {, }, [, ]` as valid delimiters for words.
// So, "word.word-word+word:word;word_word~word word(word)word{word}[word]"
// would be converted to WordWordWordWordWordWordWordWordWordWordWordWordWord
func CamelCaseEx(str string) string {
	s := strings.Trim(str, " ")

	n := ""
	capNext := true
	for _, v := range s {
		if unicode.IsUpper(v) {
			n += string(v)
		}
		if unicode.IsDigit(v) {
			n += string(v)
		}
		if unicode.IsLower(v) {
			if capNext {
				n += strings.ToUpper(string(v))
			} else {
				n += string(v)
			}
		}
		_, capNext = separatorSet[v]
	}
	return n
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

// Camel converts the given name into a camelCase.
//
//	user_info  => userInfo
//	full_name  => fullName
//	user_id    => userID
//	full-admin => fullAdmin
func Camel(s string) string {
	words := strings.FieldsFunc(s, isSeparator)
	if len(words) == 1 {
		return strings.ToLower(words[0])
	}
	return strings.ToLower(words[0]) + pascalWords(words[1:])
}

// StringWithTypeNameToGoComment renders a possible multi-line string as a
// valid Go-Comment, including the name of the type being referenced. Each line
// is prefixed as a comment.
func StringWithTypeNameToGoComment(in, typeName string) string {
	return stringToGoCommentWithPrefix(in, typeName)
}

// StringToGoComment renders a possible multi-line string as a valid Go-Comment.
// Each line is prefixed as a comment.
func StringToGoComment(in string) string {
	return stringToGoCommentWithPrefix(in, "")
}

// PathToTypeName converts a path, like Object/field1/nestedField into a go
// type name.
func PathToTypeName(path []string) string {
	for i, p := range path {
		path[i] = CamelCaseEx(p)
	}
	return strings.Join(path, "_")
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
	var (
		j int
		b strings.Builder
	)
	for i := 0; i < len(s); i++ {
		r := rune(s[i])
		// Put '_' if it is not a start or end of a word, current letter is uppercase,
		// and previous is lowercase (cases like: "UserInfo"), or next letter is also
		// a lowercase and previous letter is not "_".
		if i > 0 && i < len(s)-1 && unicode.IsUpper(r) {
			if unicode.IsLower(rune(s[i-1])) ||
				j != i-1 && unicode.IsLower(rune(s[i+1])) && unicode.IsLetter(rune(s[i-1])) {
				j = i
				b.WriteString("_")
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// HasField determines if a struct has a field with the given name.
func HasField(v any, name string) bool {
	vr := reflect.Indirect(reflect.ValueOf(v))
	return vr.FieldByName(name).IsValid()
}

func isSeparator(r rune) bool {
	return r == '_' || r == '-' || unicode.IsSpace(r)
}

func PascalWords(words []string) string {
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

// Pascal converts the given name into a PascalCase.
//
//	user_info 	=> UserInfo
//	full_name 	=> FullName
//	user_id   	=> UserID
//	full-admin	=> FullAdmin
func Pascal(s string) string {
	words := strings.FieldsFunc(s, isSeparator)
	return PascalWords(words)
}

// Receiver returns the receiver name of the given type.
//
//	[]T       => t
//	[1]T      => t
//	User      => u
//	UserQuery => uq
func Receiver(s string, importPkg map[string]string) (r string) {
	// Trim invalid tokens for identifier prefix.
	s = strings.Trim(s, "[]*&0123456789")
	parts := strings.Split(Snake(s), "_")
	min := len(parts[0])
	for _, w := range parts[1:] {
		if len(w) < min {
			min = len(w)
		}
	}
	for i := 1; i < min; i++ {
		r := parts[0][:i]
		for _, w := range parts[1:] {
			r += w[:i]
		}
		if _, ok := importPkg[r]; !ok {
			s = r
			break
		}
	}
	name := strings.ToLower(s)
	if token.Lookup(name).IsKeyword() {
		name = "_" + name
	}
	return name
}

// Quote only strings.
func Quote(v any) any {
	if s, ok := v.(string); ok {
		return strconv.Quote(s)
	}
	return v
}

// Join is a wrapper around strings.Join to provide consistent output.
func Join(a []string, sep string) string {
	sort.Strings(a)
	return strings.Join(a, sep)
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
