package handler

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/textproto"
	"strings"
)

const (
	// extractorLimit is arbitrary number to limit values extractor can return. this limits possible resource exhaustion
	// attack vector
	extractorLimit = 20
)

var errHeaderExtractorValueMissing = errors.New("missing value in request header")
var errHeaderExtractorValueInvalid = errors.New("invalid value in request header")
var errQueryExtractorValueMissing = errors.New("missing value in the query string")
var errParamExtractorValueMissing = errors.New("missing value in path params")
var errCookieExtractorValueMissing = errors.New("missing value in cookies")
var errFormExtractorValueMissing = errors.New("missing value in the form")

// ValuesExtractor defines a function for extracting values (keys/tokens) from the given context.
type ValuesExtractor func(c *gin.Context) ([]string, error)

// CreateExtractors creates a list of extractors based on the given list of extractor names.
func CreateExtractors(lookups string, authScheme string) ([]ValuesExtractor, error) {
	if lookups == "" {
		return nil, nil
	}
	sources := strings.Split(lookups, ",")
	var extractors = make([]ValuesExtractor, 0)
	for _, source := range sources {
		parts := strings.Split(source, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("extractor source for lookup could not be split into needed parts: %v", source)
		}

		switch parts[0] {
		case "query":
			extractors = append(extractors, ValuesFromQuery(parts[1]))
		case "param":
			extractors = append(extractors, ValuesFromParam(parts[1]))
		case "cookie":
			extractors = append(extractors, ValuesFromCookie(parts[1]))
		case "form":
			extractors = append(extractors, ValuesFromForm(parts[1]))
		case "header":
			prefix := ""
			if len(parts) > 2 {
				prefix = parts[2]
			} else if authScheme != "" && parts[1] == "Authorization" {
				// backwards compatibility for JWT and KeyAuth:
				// * we only apply this fix to Authorization as header we use and uses prefixes like "Bearer <token-value>" etc
				// * previously header extractor assumed that auth-scheme/prefix had a space as suffix we need to retain that
				//   behaviour for default values and Authorization header.
				prefix = authScheme
				if !strings.HasSuffix(prefix, " ") {
					prefix += " "
				}
			}
			extractors = append(extractors, ValuesFromHeader(parts[1], prefix))
		}
	}
	if len(extractors) == 0 {
		return nil, fmt.Errorf("no extractors created from lookup sources: %s %s", lookups, authScheme)
	}
	return extractors, nil
}

// ValuesFromHeader returns a functions that extracts values from the request header.
// valuePrefix is parameter to remove first part (prefix) of the extracted value. This is useful if header value has static
// prefix like `Authorization: <auth-scheme> <authorisation-parameters>` where part that we want to remove is `<auth-scheme> `
// note the space at the end. In case of basic authentication `Authorization: Basic <credentials>` prefix we want to remove
// is `Basic `. In case of NewJWT tokens `Authorization: Bearer <token>` prefix is `Bearer `.
// If prefix is left empty the whole value is returned.
func ValuesFromHeader(header string, valuePrefix string) ValuesExtractor {
	prefixLen := len(valuePrefix)
	// standard library parses http.Request header keys in canonical form but we may provide something else so fix this
	header = textproto.CanonicalMIMEHeaderKey(header)
	return func(c *gin.Context) ([]string, error) {
		values := c.Request.Header.Values(header)
		if len(values) == 0 {
			return nil, errHeaderExtractorValueMissing
		}

		result := make([]string, 0)
		for i, value := range values {
			if prefixLen == 0 {
				result = append(result, value)
				if i >= extractorLimit-1 {
					break
				}
				continue
			}
			if len(value) > prefixLen && strings.EqualFold(value[:prefixLen], valuePrefix) {
				result = append(result, value[prefixLen:])
				if i >= extractorLimit-1 {
					break
				}
			}
		}

		if len(result) == 0 {
			if prefixLen > 0 {
				return nil, errHeaderExtractorValueInvalid
			}
			return nil, errHeaderExtractorValueMissing
		}
		return result, nil
	}
}

// ValuesFromQuery returns a function that extracts values from the query string.
func ValuesFromQuery(param string) ValuesExtractor {
	return func(c *gin.Context) ([]string, error) {
		result := c.QueryArray(param)
		if len(result) == 0 {
			return nil, errQueryExtractorValueMissing
		} else if len(result) > extractorLimit-1 {
			result = result[:extractorLimit]
		}
		return result, nil
	}
}

// ValuesFromParam returns a function that extracts values from the url param string.
func ValuesFromParam(param string) ValuesExtractor {
	return func(c *gin.Context) ([]string, error) {
		result := make([]string, 0)
		for i, p := range c.Params {
			if param == p.Key {
				result = append(result, p.Value)
				if i >= extractorLimit-1 {
					break
				}
			}
		}
		if len(result) == 0 {
			return nil, errParamExtractorValueMissing
		}
		return result, nil
	}
}

// ValuesFromCookie returns a function that extracts values from the named cookie.
func ValuesFromCookie(name string) ValuesExtractor {
	return func(c *gin.Context) ([]string, error) {
		cookies := c.Request.Cookies()
		if len(cookies) == 0 {
			return nil, errCookieExtractorValueMissing
		}

		result := make([]string, 0)
		for i, cookie := range cookies {
			if name == cookie.Name {
				result = append(result, cookie.Value)
				if i >= extractorLimit-1 {
					break
				}
			}
		}
		if len(result) == 0 {
			return nil, errCookieExtractorValueMissing
		}
		return result, nil
	}
}

// ValuesFromForm returns a function that extracts values from the form field.
func ValuesFromForm(name string) ValuesExtractor {
	return func(c *gin.Context) ([]string, error) {
		if c.Request.Form == nil {
			_ = c.Request.ParseMultipartForm(32 << 20) // same what `c.Request().FormValue(name)` does
		}
		values := c.Request.Form[name]
		if len(values) == 0 {
			return nil, errFormExtractorValueMissing
		}
		if len(values) > extractorLimit-1 {
			values = values[:extractorLimit]
		}
		result := append([]string{}, values...)
		return result, nil
	}
}
