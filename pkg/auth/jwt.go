package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/url"
	"os"
	"reflect"
	"strings"
)

const (
	AlgorithmHS256 = "HS256"
)

var (
	ErrJWTMissing = errors.New("missing or malformed jwt")
	ErrJWTInvalid = errors.New("invalid or expired jwt")
)

type (
	JWTOptions struct {
		// ContinueOnIgnoredError allows the next middleware/handler to be called when ErrorHandlerWithContext decides to
		// ignore the error (by returning `nil`).
		// This is useful when parts of your site/api allow public access and some authorized routes provide extra functionality.
		// In that case you can use ErrorHandlerWithContext to set a default public JWT token value in the request context
		// and continue. Some logic down the remaining execution chain needs to check that (public) token value then.
		ContinueOnIgnoredError bool

		// Signing key to validate token.
		// This is one of the three options to provide a token validation key.
		// The order of precedence is a user-defined KeyFunc, SigningKeys and SigningKey.
		// Required if neither user-defined KeyFunc nor SigningKeys is provided.
		SigningKey interface{}

		// Map of signing keys to validate token with kid field usage.
		// This is one of the three options to provide a token validation key.
		// The order of precedence is a user-defined KeyFunc, SigningKeys and SigningKey.
		// Required if neither user-defined KeyFunc nor SigningKey is provided.
		SigningKeys map[string]interface{}

		// Signing method used to check the token's signing algorithm.
		// Optional. Default value HS256.
		SigningMethod string

		// Context key to store user information from the token into context.
		// Optional. Default value "user".
		ContextKey string

		// Claims are extendable claims data defining token content. Used by default ParseTokenFunc implementation.
		// Not used if custom ParseTokenFunc is set.
		// Optional. Default value jwt.MapClaims
		Claims jwt.Claims

		// TokenLookup is a string in the form of "<source>:<name>" or "<source>:<name>,<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>" or "header:<name>:<cut-prefix>"
		// 			`<cut-prefix>` is argument value to cut/trim prefix of the extracted value. This is useful if header
		//			value has static prefix like `Authorization: <auth-scheme> <authorisation-parameters>` where part that we
		//			want to cut is `<auth-scheme> ` note the space at the end.
		//			In case of JWT tokens `Authorization: Bearer <token>` prefix we cut is `Bearer `.
		// If prefix is left empty the whole value is returned.
		// - "query:<name>"
		// - "param:<name>"
		// - "cookie:<name>"
		// - "form:<name>"
		// Multiple sources example:
		// - "header:Authorization,cookie:myowncookie"
		TokenLookup string

		// AuthScheme to be used in the Authorization header.
		// Optional. Default value "Bearer".
		AuthScheme string

		// KeyFunc defines a user-defined function that supplies the public key for a token validation.
		// The function shall take care of verifying the signing algorithm and selecting the proper key.
		// A user-defined KeyFunc can be useful if tokens are issued by an external party.
		// Used by default ParseTokenFunc implementation.
		//
		// When a user-defined KeyFunc is provided, SigningKey, SigningKeys, and SigningMethod are ignored.
		// This is one of the three options to provide a token validation key.
		// The order of precedence is a user-defined KeyFunc, SigningKeys and SigningKey.
		// Required if neither SigningKeys nor SigningKey is provided.
		// Not used if custom ParseTokenFunc is set.
		// Default to an internal implementation verifying the signing algorithm and selecting the proper key.
		KeyFunc jwt.Keyfunc

		// ParseTokenFunc defines a user-defined function that parses token from given auth. Returns an error when token
		// parsing fails or parsed token is invalid.
		// Defaults to implementation using `github.com/golang-jwt/jwt` as JWT implementation library
		ParseTokenFunc func(auth string, ctx context.Context) (interface{}, error)
	}
)

var (
	defaultJWTOptions = JWTOptions{
		SigningMethod: AlgorithmHS256,
		ContextKey:    "user",
		TokenLookup:   "header:Authorization",
		AuthScheme:    "Bearer",
		Claims:        jwt.MapClaims{},
		KeyFunc:       nil,
	}
)

func NewJWT() *JWTOptions {
	v := defaultJWTOptions
	return &v
}

func (opts *JWTOptions) defaultParseToken(authStr string, ctx context.Context) (interface{}, error) {
	var err error

	var token *jwt.Token
	// Issue #647, #656
	if _, ok := opts.Claims.(jwt.MapClaims); ok {
		token, err = jwt.Parse(authStr, opts.KeyFunc)
	} else {
		t := reflect.ValueOf(opts.Claims).Type().Elem()
		claims := reflect.New(t).Interface().(jwt.Claims)
		token, err = jwt.ParseWithClaims(authStr, claims, opts.KeyFunc)
	}
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrJWTInvalid
	}
	return token, nil
}

// defaultKeyFunc returns a signing key of the given token.
func (opts *JWTOptions) defaultKeyFunc(t *jwt.Token) (interface{}, error) {
	// Check the signing method
	if t.Method.Alg() != opts.SigningMethod {
		return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
	}
	if len(opts.SigningKeys) > 0 {
		if kid, ok := t.Header["kid"].(string); ok {
			if key, ok := opts.SigningKeys[kid]; ok {
				return key, nil
			}
		}
		return nil, fmt.Errorf("unexpected jwt key id=%v", t.Header["kid"])
	}

	return opts.SigningKey, nil
}

func (opts *JWTOptions) Apply() error {
	if opts.SigningKey == nil && len(opts.SigningKeys) == 0 && opts.KeyFunc == nil && opts.ParseTokenFunc == nil {
		return fmt.Errorf("jwt middleware requires signing key")
	}
	if sk, ok := opts.SigningKey.(string); ok {
		switch strings.ToUpper(opts.SigningMethod) {
		case "RS256", "RS384", "RS512":
			if strings.HasPrefix(sk, "file://") {
				uri, err := url.Parse(sk)
				if err != nil {
					return err
				}
				path := conf.Abs(uri.Path)
				fc, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				opts.SigningKey, err = jwt.ParseRSAPublicKeyFromPEM(fc)
				if err != nil {
					return err
				}
			} else {
				var err error
				opts.SigningKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(sk))
				if err != nil {
					return err
				}
			}
		case "ES256", "ES384", "ES512":
		case "PS256", "PS384", "PS512":
		case "HS256", "HS384", "HS512":
			opts.SigningKey = []byte(sk)
		default:
			return fmt.Errorf("jwt middleware requires signing method")
		}
	}
	if opts.KeyFunc == nil {
		opts.KeyFunc = opts.defaultKeyFunc
	}
	if opts.ParseTokenFunc == nil {
		opts.ParseTokenFunc = opts.defaultParseToken
	}
	return nil
}
