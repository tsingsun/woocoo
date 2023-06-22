package security

import (
	"context"

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

func (i *GenericIdentity) Name() string {
	return i.claims["sub"].(string)
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
func GenericIdentityFromContext(ctx context.Context) *GenericIdentity {
	gp, _ := ctx.Value(userContextKey).(*GenericPrincipal)
	return gp.GenericIdentity
}

// GenericPrincipalFromContext get the user from context
func GenericPrincipalFromContext(ctx context.Context) *GenericPrincipal {
	gp, _ := ctx.Value(userContextKey).(*GenericPrincipal)
	return gp
}
