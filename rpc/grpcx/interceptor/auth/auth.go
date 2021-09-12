package auth

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/jwttool"
	"github.com/tsingsun/woocoo/pkg/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
	// ErrEmptyAuthHeader can be thrown if authing with a HTTP header, the auth header needs to be set
	errEmptyAuthHeader = status.Errorf(codes.InvalidArgument, "auth header is empty")
	// ErrInvalidAuthHeader indicates auth header is invalid, could for example have the wrong Realm name
	errInvalidAuthHeader = status.Errorf(codes.InvalidArgument, "auth header is invalid")
	//IdentityHandler provide an access for customize user handler
	IdentityHandler func(ctx context.Context, claims jwt.MapClaims) user.Identity
)

type authIncomingKey struct{}

func UnaryServerInterceptor(cfg *conf.Configuration) grpc.UnaryServerInterceptor {
	a, err := New()
	if err != nil {
		panic(err)
	}
	a.Apply(cfg)
	return a.UnaryServerInterceptor()
}

type Auth struct {
	jwttool.JwtParser
	TenantHeader string
	// Set the identity handler function
	IdentityHandler func(ctx context.Context, claims jwt.MapClaims) user.Identity
}

func New() (*Auth, error) {
	identityKey := "sub"
	a := &Auth{
		JwtParser: jwttool.JwtParser{
			Realm:            "woocoo",
			IdentityKey:      identityKey,
			TokenLookup:      "header:authorization",
			SigningAlgorithm: "HS256",
			Timeout:          time.Hour,
			TokenHeadName:    "Bearer",
			TimeFunc:         time.Now,
		},
	}

	return a, nil
}

func (a *Auth) Apply(cfg *conf.Configuration) {

	if err := cfg.Parser().UnmarshalByJson("", a); err != nil {
		panic(err)
	}
	if a.PrivKeyFile != "" {
		a.PrivKeyFile = cfg.Abs(a.PrivKeyFile)
	}
	if a.PubKeyFile != "" {
		a.PubKeyFile = cfg.Abs(a.PubKeyFile)
	}
	a.Key = []byte(cfg.String("secret"))
	a.AuthInit()
}

func (a *Auth) AuthInit() {
	if IdentityHandler != nil {
		a.IdentityHandler = IdentityHandler
	}

	if a.IdentityHandler == nil {
		a.IdentityHandler = func(ctx context.Context, claims jwt.MapClaims) user.Identity {
			return &user.User{"id": claims[a.IdentityKey]}
		}
	}
}

func (a *Auth) getTokenOriStr(md metadata.MD) jwttool.GetToken {

	return func(fromType string, key string) (string, error) {
		var token string
		var err error
		switch fromType {
		case "header":
			token, err = a.jwtFromMetadata(md, key)
		}
		return token, err
	}
}

func (a *Auth) jwtFromMetadata(md metadata.MD, key string) (string, error) {
	authHeader, ok := md[key]

	if !ok {
		return "", errEmptyAuthHeader
	}

	parts := strings.SplitN(authHeader[0], " ", 2)
	if !(len(parts) == 2 && parts[0] == a.TokenHeadName) {
		return "", errInvalidAuthHeader
	}

	return parts[1], nil
}

func (a *Auth) GetClaimsFromJWT(md metadata.MD) (jwt.MapClaims, error) {
	token, err := a.ParseToken(a.getTokenOriStr(md), nil)
	if err != nil {
		return nil, err
	}
	claims := jwt.MapClaims{}
	for key, value := range token.Claims.(jwt.MapClaims) {
		claims[key] = value
	}
	return claims, nil
}

func (a *Auth) validator(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}
	claims, err := a.GetClaimsFromJWT(md)
	if err != nil {
		return ctx, err
	}
	if claims["exp"] == nil {
		return ctx, errInvalidToken
	}

	if _, ok := claims["exp"].(float64); !ok {
		return ctx, errInvalidToken
	}
	if int64(claims["exp"].(float64)) < a.TimeFunc().Unix() {
		return ctx, errInvalidToken
	}

	// try set orgID
	if org, ok := md[user.OrgIdHeader]; ok {
		claims[user.OrgIdHeader] = org
	}
	identity := a.IdentityHandler(ctx, claims)
	if identity != nil {
		return AppendToContext(ctx, identity), nil
	}
	return ctx, nil
}

func AppendToContext(ctx context.Context, identity user.Identity) context.Context {
	return context.WithValue(ctx, authIncomingKey{}, identity)
}

func FromIncomingContext(ctx context.Context) (user.Identity, bool) {
	md, ok := ctx.Value(authIncomingKey{}).(user.Identity)
	if !ok {
		return nil, false
	}
	return md, true
}

// UnaryServerInterceptor ensures a valid token exists within a request's metadata. If
// the token is missing or invalid, the interceptor blocks execution of the
// handler and returns an error. Otherwise, the interceptor invokes the unary
// handler.
func (a *Auth) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// The keys within metadata.MD are normalized to lowercase.
		// See: https://godoc.org/google.golang.org/grpc/metadata#New
		newctx, err := a.validator(ctx)
		if err != nil {
			return nil, err
		}
		// Continue execution of handler after ensuring a valid token.
		return handler(newctx, req)
	}
}
