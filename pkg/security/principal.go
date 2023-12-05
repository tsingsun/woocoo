package security

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// PrincipalContextKey is the key of context which store the user.
	PrincipalContextKey = "woocoo_user"
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
	// A Principal is typically defined as an entity that has a unique identifier in a security context.
	// It can be a user, computer, process, service, or any other entity. In the context of security,
	// a Principal represents an entity that can operate independently or participate in security-related activities.
	//
	// A principal object represents the security context of the user on whose behalf the code is running,
	// including that user's identity (IIdentity) and any roles to which they belong, but now only identity.
	Principal interface {
		Identity() Identity
	}
)

// WithContext Add user to context.
func WithContext(ctx context.Context, user Principal) context.Context {
	return context.WithValue(ctx, PrincipalContextKey, user) // nolint: staticcheck
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(PrincipalContextKey).(Principal)
	return p, ok
}
