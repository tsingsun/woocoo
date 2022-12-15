package interceptor

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
)

// ValuesExtractor defines a function for extracting values (keys/tokens) from the given context.
type ValuesExtractor func(ctx context.Context) ([]string, error)

type (
	jwtKey struct{}
	// JWTOptions is the options for JWT interceptor.
	JWTOptions struct {
		auth.JWTOptions `json:",squash" yaml:",squash"`
	}
	// JWT is the interceptor for JWT.
	JWT struct {
	}
)

func NewJWTOptions() *JWTOptions {
	return &JWTOptions{
		JWTOptions: *auth.NewJWT(),
	}
}

func (mw *JWTOptions) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(&mw); err != nil {
		panic(err)
	}
	if err := mw.JWTOptions.Apply(); err != nil {
		panic(err)
	}
}

// JWTFromIncomingContext extracts the token from the incoming context which `ParseTokenFunc` used default token parser.
func JWTFromIncomingContext(ctx context.Context) (*jwt.Token, bool) {
	token, ok := ctx.Value(jwtKey{}).(*jwt.Token)
	if !ok {
		return nil, false
	}
	return token, true
}

// Name returns the name of the interceptor.
func (JWT) Name() string {
	return "jwt"
}

// SteamServerInterceptor jwt ServerInterceptor for stream server.
func (JWT) SteamServerInterceptor(cfg *conf.Configuration) grpc.StreamServerInterceptor {
	options := NewJWTOptions()
	options.Apply(cfg)
	var extractor = createExtractor(options)
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		authstr, err := extractor(ss.Context())
		if err != nil {
			return auth.ErrJWTMissing
		}
		var lastTokenErr error
		for _, s := range authstr {
			token, err := options.ParseTokenFunc(ss.Context(), s)
			if err != nil {
				lastTokenErr = err
				continue
			}
			newctx := context.WithValue(ss.Context(), jwtKey{}, token)
			ws := WrapServerStream(ss)
			ws.WrappedContext = newctx
			return handler(srv, ws)
		}
		if err != nil {
			return err
		}
		if lastTokenErr != nil {
			return err
		}
		// Continue execution of handler after ensuring a valid token.
		return auth.ErrJWTMissing
	}
}

// UnaryServerInterceptor ensures a valid token exists within a request's metadata. If
// the token is missing or invalid, the interceptor blocks execution of the
// handler and returns an error. Otherwise, the interceptor invokes the unary
// handler.
func (JWT) UnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	interceptor := NewJWTOptions()
	interceptor.Apply(cfg)
	var extractor = createExtractor(interceptor)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// The keys within metadata.MD are normalized to lowercase.
		// See: https://godoc.org/google.golang.org/grpc/metadata#New
		authstr, err := extractor(ctx)
		if err != nil {
			return nil, auth.ErrJWTMissing
		}
		var lastTokenErr error
		for _, s := range authstr {
			token, err := interceptor.ParseTokenFunc(ctx, s)
			if err != nil {
				lastTokenErr = err
				continue
			}
			newctx := context.WithValue(ctx, jwtKey{}, token)
			return handler(newctx, req)
		}
		if err != nil {
			return nil, err
		}
		if lastTokenErr != nil {
			return nil, err
		}
		// Continue execution of handler after ensuring a valid token.
		return nil, auth.ErrJWTMissing
	}
}

func valuesFromMetadata(key string, valuePrefix string) ValuesExtractor {
	prefixLen := len(valuePrefix)
	return func(ctx context.Context) ([]string, error) {
		md, ok := metadata.FromIncomingContext(ctx)

		if !ok {
			return nil, auth.ErrJWTMissing
		}
		values, ok := md[key]

		if !ok {
			return nil, auth.ErrJWTMissing
		}
		result := make([]string, 0)
		for _, value := range values {
			if prefixLen == 0 {
				result = append(result, value)
				continue
			}
			if len(value) > prefixLen && strings.EqualFold(value[:prefixLen], valuePrefix) {
				result = append(result, value[prefixLen:])
			}
		}

		if len(result) == 0 {
			if prefixLen > 0 {
				return nil, fmt.Errorf("invalid value in request metadata")
			}
			return nil, fmt.Errorf("missing value in request metadata")
		}
		return result, nil
	}
}

func createExtractor(interceptor *JWTOptions) func(ctx context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		prefix := ""
		if interceptor.AuthScheme != "" {
			prefix = interceptor.AuthScheme
			if !strings.HasSuffix(prefix, " ") {
				prefix += " "
			}
		}
		return valuesFromMetadata(interceptor.TokenLookup, prefix)(ctx)
	}
}
