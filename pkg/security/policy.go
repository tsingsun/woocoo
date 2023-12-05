package security

import (
	"context"
)

var (
	DefaultAuthorizer Authorizer = &noopAuthorizer{}
)

// Authorizer defines the interface for authorization.
//
// The authorization is based on system user operate the application resource, the behavior of resource can be classified as
// some actions. The authorizer check the user has permission to access the resource by the action.
type Authorizer interface {
	// Prepare accepts input infos and build EvalArgs.
	// when you implement this method, you must know the meaning of the arnParts and convert to correct Action.
	// In Web authz, the parts format is [appCode, Request.Method, Request.URL.Path] by default;
	// In Graphql, the parts format is [appCode, Request.Method, Operator.Name] by default;
	Prepare(ctx context.Context, kind ArnKind, parts ...string) (*EvalArgs, error)
	// Eval returns true if the request is allowed, otherwise returns false.
	Eval(ctx context.Context, args *EvalArgs) (bool, error)
	// QueryAllowedResourceConditions parse the conditions part of resources eval passed by resource prefix,
	// the result is a list of resource conditions that adapt Authorizer instance.
	// This method call sense: Orm need to filter the data that the user has permission to access.
	// For example, the resource arn "oss:bucket:user/1" means that the user has permission to access the `user` table data
	// whose condition is "user = 1",the result of this method is ["user = 1"].
	// You should implement complex condition by your sense if you when to filter data.
	QueryAllowedResourceConditions(ctx context.Context, args *EvalArgs) ([]string, error)
}

// EvalArgs is the request for authorization.
type EvalArgs struct {
	// User is the user who performs the operation. If you can't get the user from the context, you can set it.
	User Principal
	// Action is the operation that the user performs on the resource.
	Action Action
	// ActionVerb is the operation verb that may be in part of action.
	//
	// For example, the action is "user:createUser", the verb is "create". The verb is empty in most cases.
	// If the Authorizer implement user a verb in policy such as casbin, a policy ['p','/user','read']
	ActionVerb string
	// Resource is the resource that the user performs the operation on, empty in most cases.
	Resource Resource
}

// IsAllowed checks if the user has permission to do an operation on a resource. It uses the default authorizer, so you must
// set the default authorizer before use this method, see SetDefaultAuthorizer.
func IsAllowed(ctx context.Context, kind ArnKind, parts ...string) (bool, error) {
	args, err := DefaultAuthorizer.Prepare(ctx, kind, parts...)
	if err != nil {
		return false, err
	}
	return DefaultAuthorizer.Eval(ctx, args)
}

// Action describe a resource operation.
// Action should be easy to understand like "user:createXXX", "user:updateXXX", "user:deleteXXX", "user:listXXXX".
type Action string

// MatchResource checks if the resource matches the resource pattern.
// supports  '*' and '?' wildcards in the pattern string.
func (a Action) MatchResource(resource string) bool {
	return arnMatch(string(a), resource)
}

// ArnKind define the application resource name(arn) request kind.
// application resource can be an action described by uri; data resource and so on.
type ArnKind string

const (
	// ArnKindWeb web request kind.
	ArnKindWeb ArnKind = "web"
	ArnKindRpc ArnKind = "rpc"
	ArnKindGql ArnKind = "gql"
)

// Resource is for describing a resource pattern by string expression.
// The resource can be a string or a wildcard.
// identify a resource like "oss:bucket/object", "oss:bucket/*", "oss:bucket/object/*".
type Resource string

// MatchResource checks if the resource matches the resource pattern.
// supports  '*' and '?' wildcards in the pattern string.
func (r Resource) MatchResource(resource string) bool {
	return arnMatch(string(r), resource)
}

func arnMatch(pattern, resource string) bool {
	if pattern == "" {
		return pattern == resource
	}
	if pattern == "*" {
		return true
	}
	return deepMatchRune([]rune(resource), []rune(pattern), false)
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

// noopAuthorizer is an empty implement of Authorizer, default authorizer.
type noopAuthorizer struct{}

func (d noopAuthorizer) Prepare(context.Context, ArnKind, ...string) (*EvalArgs, error) {
	return nil, nil
}

func (d noopAuthorizer) Eval(context.Context, *EvalArgs) (bool, error) {
	return true, nil
}

func (d noopAuthorizer) QueryAllowedResourceConditions(context.Context, *EvalArgs) ([]string, error) {
	return nil, nil
}

// SetDefaultAuthorizer sets the default authorization.
func SetDefaultAuthorizer(au Authorizer) {
	DefaultAuthorizer = au
}
