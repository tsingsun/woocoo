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
		auth.JWTOptions `json:",inline" yaml:",inline"`
	}
	// JWT is the interceptor for JWT.
	JWT struct {
	}
)

func NewJWTOptions() *JWTOptions {
	return &JWTOptions{
		JWTOptions: *auth.NewJWTOptions(),
	}
}

func (mw *JWTOptions) Apply(cfg *conf.Configuration) {
	if err := cfg.Unmarshal(&mw); err != nil {
		panic(err)
	}
	if err := mw.JWTOptions.Init(); err != nil {
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

// UnaryServerInterceptor ensures a valid token exists within a request's metadata. If
// the token is missing or invalid, the interceptor blocks execution of the
// handler and returns an error. Otherwise, the interceptor invokes the unary
// handler.
func (itcp JWT) UnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	options := NewJWTOptions()
	options.Apply(cfg)
	var extractor = createExtractor(options)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// The keys within metadata.MD are normalized to lowercase.
		// See: https://godoc.org/google.golang.org/grpc/metadata#New
		newctx, err := itcp.withToken(extractor, options, ctx)
		if err != nil {
			return nil, err
		}
		return handler(newctx, req)
	}
}

// SteamServerInterceptor jwt ServerInterceptor for stream server.
func (itcp JWT) SteamServerInterceptor(cfg *conf.Configuration) grpc.StreamServerInterceptor {
	options := NewJWTOptions()
	options.Apply(cfg)
	var extractor = createExtractor(options)
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		newctx, err := itcp.withToken(extractor, options, ss.Context())
		if err != nil {
			return err
		}
		ws := WrapServerStream(ss)
		ws.WrappedContext = newctx
		return handler(srv, ws)
	}
}

func (JWT) withToken(extractor ValuesExtractor, options *JWTOptions, ctx context.Context) (context.Context, error) {
	authstr, err := extractor(ctx)
	if err != nil {
		return nil, auth.ErrJWTMissing
	}
	var lastTokenErr error
	for _, s := range authstr {
		token, err := options.ParseTokenFunc(ctx, s)
		if err != nil {
			lastTokenErr = err
			continue
		}
		newctx := context.WithValue(ctx, jwtKey{}, token)
		return newctx, nil
	}
	if err != nil {
		return nil, err
	}
	if lastTokenErr != nil {
		return nil, lastTokenErr
	}
	// Continue execution of handler after ensuring a valid token.
	return nil, auth.ErrJWTMissing
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
