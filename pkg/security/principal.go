package security

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// UserContextKey is the key of context which store the user.
	UserContextKey = "woocoo_user"
)

type (
	// Identity defines the basic functionality of an identity object.
	//
	// An identity object represents the user on whose behalf the code is running
	Identity interface {
		// Name returns the identity of the user from Claims.
		// for example, the primary key of the user record in database.
		// The identity field in business system may int or string, translate it to string.
		Name() string
		// Claims uses jwt Claims for easy to get user info and declare use jwt to pass identity info.
		Claims() jwt.Claims
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
	return context.WithValue(ctx, UserContextKey, user) // nolint: staticcheck
}
