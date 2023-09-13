package security

import (
	"context"
	"strconv"

	"github.com/golang-jwt/jwt/v4"
)

type (
	// GenericPrincipal Represents a generic principal.
	GenericPrincipal struct {
		GenericIdentity *GenericIdentity
	}
	// GenericIdentity Represents a generic user.
	GenericIdentity struct {
		name   string
		claims jwt.MapClaims
	}
)

func (p *GenericPrincipal) Identity() Identity {
	return p.GenericIdentity
}

// Name returns the id of the user.
func (i *GenericIdentity) Name() string {
	return i.claims["sub"].(string)
}

// NameInt returns the id of the user. if not int, return 0
func (i *GenericIdentity) NameInt() int {
	s := i.claims["sub"].(string)
	id, _ := strconv.Atoi(s)
	return id
}

func (i *GenericIdentity) Claims() jwt.MapClaims {
	return i.claims
}

// NewGenericPrincipalByClaims return GenericPrincipal
func NewGenericPrincipalByClaims(claims jwt.MapClaims) *GenericPrincipal {
	return &GenericPrincipal{
		GenericIdentity: &GenericIdentity{claims: claims},
	}
}

// GenericIdentityFromContext get the user from context
func GenericIdentityFromContext(ctx context.Context) (*GenericIdentity, bool) {
	gp, ok := ctx.Value(UserContextKey).(*GenericPrincipal)
	if !ok {
		return nil, false
	}
	return gp.GenericIdentity, ok
}

// GenericPrincipalFromContext get the user from context
func GenericPrincipalFromContext(ctx context.Context) (*GenericPrincipal, bool) {
	gp, ok := ctx.Value(UserContextKey).(*GenericPrincipal)
	return gp, ok
}
