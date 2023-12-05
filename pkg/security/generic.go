package security

import (
	"github.com/golang-jwt/jwt/v5"
)

type (
	// GenericPrincipal Represents a generic principal.
	GenericPrincipal struct {
		GenericIdentity *GenericIdentity
	}
	// GenericIdentity Represents a generic user.
	GenericIdentity struct {
		claims jwt.Claims
	}
)

func (p *GenericPrincipal) Identity() Identity {
	return p.GenericIdentity
}

// Name returns the id of the user if any.
func (i *GenericIdentity) Name() string {
	v, _ := i.claims.GetSubject()
	return v
}

func (i *GenericIdentity) Claims() jwt.Claims {
	return i.claims
}

// NewGenericPrincipalByClaims return GenericPrincipal
func NewGenericPrincipalByClaims(claims jwt.Claims) *GenericPrincipal {
	return &GenericPrincipal{
		GenericIdentity: &GenericIdentity{claims: claims},
	}
}
