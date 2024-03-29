package httpx

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// ExtractorLimit is arbitrary number to limit values extractor can return. this limits possible resource exhaustion
	// attack vector
	ExtractorLimit = 20
)

var (
	errHeaderExtractorValueMissing = errors.New("missing value in request header")
	errHeaderExtractorValueInvalid = errors.New("invalid value in request header")
)

// ValuesFromHeader returns functions that extract values from the request header.
// valuePrefix is a parameter to remove the first part (prefix) of the extracted value. This is useful if header value has static
// prefix like `Authorization: <auth-scheme> <authorisation-parameters>` where part that we want to remove is `<auth-scheme> `
// note the space at the end. In the case of basic authentication `Authorization: Basic <credentials>` prefix we want to remove
// is `Basic `. In the case of NewJWT tokens `Authorization: Bearer <token>` prefix is `Bearer `.
// If the prefix is left empty, the whole value is returned.
func ValuesFromHeader(r *http.Request, header string, valuePrefix string, prefixLen int) ([]string, error) {
	values := r.Header.Values(header)
	if len(values) == 0 {
		return nil, errHeaderExtractorValueMissing
	}

	result := make([]string, 0)
	for i, value := range values {
		if prefixLen == 0 {
			result = append(result, value)
			if i >= ExtractorLimit-1 {
				break
			}
			continue
		}
		if len(value) > prefixLen && strings.EqualFold(value[:prefixLen], valuePrefix) {
			result = append(result, value[prefixLen:])
			if i >= ExtractorLimit-1 {
				break
			}
		}
	}

	if len(result) == 0 {
		if prefixLen > 0 {
			return nil, errHeaderExtractorValueInvalid
		}
	}
	return result, nil
}

// EscapePath escapes part of a URL path in Amazon style
func EscapePath(path string, encodeSep bool) string {
	var buf bytes.Buffer
	for i := 0; i < len(path); i++ {
		c := path[i]
		if noEscape[c] || (c == '/' && !encodeSep) {
			buf.WriteByte(c)
		} else {
			fmt.Fprintf(&buf, "%%%02X", c)
		}
	}
	return buf.String()
}

func getURIPath(u *url.URL) string {
	var uri string

	if len(u.Opaque) > 0 {
		uri = "/" + strings.Join(strings.Split(u.Opaque, "/")[3:], "/")
	} else {
		uri = u.EscapedPath()
	}

	if len(uri) == 0 {
		uri = "/"
	}

	return uri
}

// ValuesFromCanonical attempts to extract the value of a canonical string.
// a canonical string is a string of key value pairs separated by deli1 and deli2
func ValuesFromCanonical(src, deli1, deli2 string) map[string]string {
	vs := make(map[string]string)
	ps := strings.Split(src, deli1)
	for _, p := range ps {
		kv := strings.SplitN(p, deli2, 2)
		if len(kv) != 2 {
			continue
		}
		vs[kv[0]] = kv[1]
	}
	return vs
}

// GetSignedRequestSignature attempts to extract the signature of the request.
// Returning an error if the request is unsigned, or unable to extract the
// signature.
func GetSignedRequestSignature(r *http.Request, header, scheme, delt string) (string, error) {
	auth, err := ValuesFromHeader(r, header, scheme, len(scheme))
	if err != nil {
		return "", err
	}
	ps := strings.Split(auth[0], delt)
	for _, p := range ps {
		if idx := strings.Index(p, signatureElem); idx >= 0 {
			sig := p[len(signatureElem):]
			if len(sig) == 0 {
				return "", fmt.Errorf("invalid request signature authorization header")
			}
			return sig, nil
		}
	}

	return "", fmt.Errorf("request not signed")
}

// FormatSignTime format time to string by layout, if layout is empty, return unix timestamp.
func FormatSignTime(t time.Time, layout string) string {
	if layout == "" {
		return strconv.Itoa(int(t.Unix()))
	}
	return t.Format(layout)
}

// ParseSignTime parse string to time by layout, if layout is empty, parse as unix timestamp.
func ParseSignTime(layout string, str string) (time.Time, error) {
	if layout == "" {
		uf, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(uf, 0), nil
	}
	return time.Parse(layout, str)
}
