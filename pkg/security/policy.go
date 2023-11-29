package security

import (
	"context"
)

const (
	ArnSplit      = ":"
	ArnBlockSplit = "/"
)

var (
	DefaultAuthorizer Authorizer = &noopAuthorizer{}
)

// Authorizer defines the interface for authorization.
//
// The authorization is based on system user operate the application resource, the behavior of resource can be classified as
// some actions. The authorizer check the user has permission to access the resource by the action.
type Authorizer interface {
	// Conv converts the input meta string to a Resource instance.
	// when you implement this method, you must know the meaning of the arnParts and convert to correct resource.
	// In Web authz, the arn parts format is [appCode, Request.Method, Request.URL.Path];
	// In Graphql, the arn parts format is [appCode, Request.Method, Operator.Name];
	Conv(kind ArnRequestKind, arnParts ...string) Resource
	// Eval returns true if the request is allowed, otherwise returns false.
	Eval(ctx context.Context, identity Identity, item Resource) (bool, error)
	// QueryAllowedResourceConditions parse the conditions part of resources eval passed by resource prefix,
	// the result is a list of resource conditions that adapt Authorizer instance.
	// This method call sense: Orm need to filter the data that the user has permission to access.
	// For example, the resource arn "oss:bucket:user/1" means that the user has permission to access the `user` table data
	// whose condition is "user = 1",the result of this method is ["user = 1"].
	// You should implement complex condition by your sense if you when to filter data.
	QueryAllowedResourceConditions(ctx context.Context, identity Identity, item Resource) ([]string, error)
}

type noopAuthorizer struct{}

func (d noopAuthorizer) Conv(kind ArnRequestKind, arnParts ...string) Resource {
	return ""
}

func (d noopAuthorizer) Eval(ctx context.Context, identity Identity, item Resource) (bool, error) {
	return true, nil
}

func (d noopAuthorizer) QueryAllowedResourceConditions(ctx context.Context, identity Identity, item Resource) ([]string, error) {
	return nil, nil
}

// SetDefaultAuthorizer sets the default authorization.
func SetDefaultAuthorizer(au Authorizer) {
	DefaultAuthorizer = au
}

// Action describe a resource operation.
// Action should be easy to understand like "user:createXXX", "user:updateXXX", "user:deleteXXX", "user:listXXXX".
type Action string

// ArnRequestKind define the application resource name request kind.
type ArnRequestKind string

const (
	// ArnRequestKindWeb web request kind.
	ArnRequestKindWeb ArnRequestKind = "web"
	ArnRequestKindRpc ArnRequestKind = "rpc"
	ArnRequestKindGql ArnRequestKind = "gql"
)

// Resource is for describing a resource pattern by string expression.
// The resource can be a string or a wildcard.
// identify a resource like "oss:bucket/object", "oss:bucket/*", "oss:bucket/object/*".
type Resource string

// MatchResource checks if the resource matches the resource pattern.
// supports  '*' and '?' wildcards in the pattern string.
func (r Resource) MatchResource(resource string) bool {
	if r == "" {
		return string(r) == resource
	}
	if r == "*" {
		return true
	}
	return deepMatchRune([]rune(resource), []rune(r), false)
}

func deepMatchRune(str, pattern []rune, simple bool) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		case '?':
			if len(str) == 0 {
				return simple
			}
		case '*':
			return deepMatchRune(str, pattern[1:], simple) ||
				(len(str) > 0 && deepMatchRune(str[1:], pattern, simple))
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}
