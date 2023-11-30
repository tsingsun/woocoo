package security

import (
	"context"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type mockAuthorizer struct{}

func (m mockAuthorizer) Conv(_ context.Context, _ ArnRequestKind, arnParts ...string) (Resource, error) {
	return Resource(strings.Join(arnParts, ArnSplit)), nil
}

func (m mockAuthorizer) Eval(ctx context.Context, identity Identity, item Resource) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockAuthorizer) QueryAllowedResourceConditions(ctx context.Context, identity Identity, item Resource) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func TestSetDefaultAuthorizer(t *testing.T) {
	SetDefaultAuthorizer(&mockAuthorizer{})
}

func TestNoopAuthorizer(t *testing.T) {
	au := noopAuthorizer{}
	r, err := au.Conv(context.Background(), ArnRequestKindWeb, "test")
	assert.NoError(t, err)
	assert.Equal(t, Resource(""), r)
	ev, err := au.Eval(context.Background(), nil, Resource(""))
	assert.NoError(t, err)
	assert.True(t, ev)
	_, err = au.QueryAllowedResourceConditions(context.Background(), nil, Resource(""))
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
