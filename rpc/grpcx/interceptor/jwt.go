package interceptor

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
)

// ValuesExtractor defines a function for extracting values (keys/tokens) from the given context.
type ValuesExtractor func(ctx context.Context) ([]string, error)

type (
	// JWTOptions is the options for JWT interceptor.
	JWTOptions struct {
		auth.JWTOptions `json:",inline" yaml:",inline"`
		// Exclude is a list of http paths to exclude from JWT auth
		//
		// path format must same as info.FullMethod started with "/".
		Exclude []string `json:"exclude" yaml:"exclude"`
	}
	// JWT is the interceptor for JWT.
	JWT struct {
	}
)

// NewJWTOptions constructs a new JWTOptions struct with supplied options.
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
		// check exclude
		if len(options.Exclude) > 0 {
			for _, e := range options.Exclude {
				if e == info.FullMethod {
					return handler(ctx, req)
				}
			}
		}
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
		// check exclude
		if len(options.Exclude) > 0 {
			for _, e := range options.Exclude {
				if e == info.FullMethod {
					return handler(srv, ss)
				}
			}
		}
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
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, auth.ErrJWTClaims
		}
		prpl := security.NewGenericPrincipalByClaims(claims)
		newctx := security.WithContext(ctx, prpl)
		return newctx, nil
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
