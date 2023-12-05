package security

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type mockAuthorizer struct {
	userNeed bool
}

func (m mockAuthorizer) Prepare(ctx context.Context, _ ArnKind, arnParts ...string) (*EvalArgs, error) {
	user, ok := FromContext(ctx)
	if !ok && m.userNeed {
		return nil, errors.New("security.IsAllow: user not found in context")
	}
	act := strings.Join(arnParts, ":")
	return &EvalArgs{
		User:   user,
		Action: Action(act),
	}, nil
}

func (m mockAuthorizer) Eval(ctx context.Context, args *EvalArgs) (bool, error) {
	if args.Action == Action("pass") {
		return true, nil
	}
	return false, nil
}

func (m mockAuthorizer) QueryAllowedResourceConditions(ctx context.Context, args *EvalArgs) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func TestIsAllowed(t *testing.T) {
	SetDefaultAuthorizer(&mockAuthorizer{})
	t.Run("allow", func(t *testing.T) {
		al, err := IsAllowed(WithContext(context.Background(), NewGenericPrincipalByClaims(jwt.MapClaims{
			"sub": "1",
		})), ArnKindWeb, "pass")
		assert.NoError(t, err)
		assert.True(t, al)
	})
	t.Run("deny", func(t *testing.T) {
		al, err := IsAllowed(WithContext(context.Background(), NewGenericPrincipalByClaims(jwt.MapClaims{
			"sub": "1",
		})), ArnKindWeb, "deny")
		assert.NoError(t, err)
		assert.False(t, al)
	})
	t.Run("miss user", func(t *testing.T) {
		SetDefaultAuthorizer(&mockAuthorizer{
			userNeed: true,
		})
		_, err := IsAllowed(context.Background(), ArnKindWeb, "test")
		assert.ErrorContains(t, err, "security.IsAllow: user not found in context")
	})
}

func TestNoopAuthorizer(t *testing.T) {
	au := noopAuthorizer{}
	r, err := au.Prepare(context.Background(), ArnKindWeb, "test")
	assert.NoError(t, err)
	assert.Nil(t, r)
	ev, err := au.Eval(context.Background(), &EvalArgs{Action: Action(""), Resource: Resource("")})
	assert.NoError(t, err)
	assert.True(t, ev)
	_, err = au.QueryAllowedResourceConditions(context.Background(), nil)
	assert.NoError(t, err)
}

func TestResource_MatchResource(t *testing.T) {
	type args struct {
		resource string
	}
	tests := []struct {
		name string
		r    Resource
		args args
		want bool
	}{
		{
			name: "empty",
			r:    Resource(""),
			args: args{
				resource: "",
			},
			want: true,
		},
		{
			name: "match all",
			r:    Resource("*"),
			args: args{
				resource: "oss:bucket/object",
			},
			want: true,
		},
		{
			name: "match",
			r:    Resource("oss:bucket/object"),
			args: args{
				resource: "oss:bucket/object",
			},
			want: true,
		},
		{
			name: "not match",
			r:    Resource("oss:bucket/object"),
			args: args{
				resource: "oss:bucket/object1",
			},
			want: false,
		},
		{
			name: "match wildcard",
			r:    Resource("oss:bucket/*"),
			args: args{
				resource: "oss:bucket/object",
			},
			want: true,
		},
		{
			name: "match '?' wildcard ",
			r:    Resource("oss:bucket/object?"),
			args: args{
				resource: "oss:bucket/object1",
			},
			want: true,
		},
		{
			name: "match '?' wildcard false ",
			r:    Resource("oss:bucket/object?"),
			args: args{
				resource: "oss:bucket/object",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.r.MatchResource(tt.args.resource), "MatchResource(%v)", tt.args.resource)
		})
	}
}
