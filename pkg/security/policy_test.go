package security

import (
	"context"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type mockAuthorizer struct{}

func (m mockAuthorizer) Conv(_ ArnRequestKind, arnParts ...string) Resource {
	return Resource(strings.Join(arnParts, ArnSplit))
}

func (m mockAuthorizer) Eval(ctx context.Context, identity Identity, item Resource) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockAuthorizer) QueryAllowedResourceConditions(ctx context.Context, identity Identity, item Resource) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func TestSetDefaultAuthorization(t *testing.T) {
	SetDefaultAuthorizer(&mockAuthorizer{})
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.r.MatchResource(tt.args.resource), "MatchResource(%v)", tt.args.resource)
		})
	}
}
