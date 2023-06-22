package security

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneric(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	type testCase struct {
		name string
		args args
		want *GenericIdentity
	}
	tests := []testCase{
		{
			name: "GenericIdentityFromContext",
			args: args{
				ctx: context.WithValue(context.Background(), userContextKey, &GenericPrincipal{
					GenericIdentity: &GenericIdentity{
						name: "test",
					},
				}),
			},
			want: &GenericIdentity{
				name: "test",
			},
		},
		{
			name: "GenericPrincipalFromContext",
			args: args{
				ctx: context.WithValue(context.Background(), userContextKey, &GenericPrincipal{
					GenericIdentity: &GenericIdentity{
						name: "test",
					},
				}),
			},
			want: &GenericIdentity{
				name: "test",
			},
		},
		{
			name: "NewGenericPrincipalByClaims",
			args: args{
				ctx: context.WithValue(context.Background(), userContextKey, NewGenericPrincipalByClaims(jwt.MapClaims{
					"sub": "test",
				})),
			},
			want: &GenericIdentity{
				name: "",
				claims: jwt.MapClaims{
					"sub": "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenericIdentityFromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got)
			got2 := GenericPrincipalFromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got2.GenericIdentity)
			if got2.Identity().Claims() != nil {
				assert.Equal(t, tt.want.Name(), got2.GenericIdentity.Name())
			}
		})
	}
}

func TestGetJtiFromToken(t *testing.T) {
	type args struct {
		ctx context.Context
		key string
	}
	type testCase struct {
		name   string
		args   args
		want   any
		wantOK bool
	}
	tests := []testCase{
		{
			name: "int jti",
			args: args{
				ctx: context.WithValue(context.Background(), "user", &jwt.Token{Claims: jwt.MapClaims{
					"sub": 1,
				}}),
				key: "user",
			},
			want:   1,
			wantOK: true,
		},
		{
			name: "NoExist",
			args: args{
				ctx: context.WithValue(context.Background(), "no", &jwt.Token{Claims: jwt.MapClaims{
					"NoExist": 1,
				}}),
				key: "user",
			},
			want:   nil,
			wantOK: false,
		},
		{
			name: "No map claims",
			args: args{
				ctx: context.WithValue(context.Background(), "user", &jwt.Token{Claims: jwt.RegisteredClaims{
					Subject: "1",
				}}),
				key: "user",
			},
			want:   nil,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetSubjectFromToken(tt.args.ctx, tt.args.key)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestWithContext(t *testing.T) {
	tests := []struct {
		name string
		user GenericPrincipal
		want GenericPrincipal
	}{
		{
			name: "set and get user",
			user: GenericPrincipal{GenericIdentity: &GenericIdentity{name: "John"}},
			want: GenericPrincipal{GenericIdentity: &GenericIdentity{name: "John"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithContext(ctx, &tt.user)
			got, ok := ctx.Value(userContextKey).(*GenericPrincipal)
			require.True(t, ok)
			assert.Equal(t, tt.want, *got)
		})
	}
}
