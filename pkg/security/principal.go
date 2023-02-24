package security

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
)

var (
	// userContextKey is the key of context which store the user.
	userContextKey struct{}
)

type (
	// Identity defines the basic functionality of an identity object.
	//
	// An identity object represents the user on whose behalf the code is running
	Identity interface {
		Name() string
		Claims() jwt.MapClaims
	}
	// Principal Defines the basic functionality of a principal object.
	//
	// A principal object represents the security context of the user on whose behalf the code is running,
	// including that user's identity (IIdentity) and any roles to which they belong.
	Principal interface {
		Identity() Identity
	}
)

// WithContext Add user to context.
func WithContext(ctx context.Context, user Principal) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetSubjectFromToken Get Sub(User) from context with Jwt default token using jwt.MapClaims.
// if not found, return nil and false.
// key is the key of context which store the jwt token.
func GetSubjectFromToken(ctx context.Context, key string) (any, bool) {
	token, ok := ctx.Value(key).(*jwt.Token)
	if !ok {
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false
	}
	v, ok := claims["sub"]
	return v, ok
}
