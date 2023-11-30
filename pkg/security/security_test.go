package security

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
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
				ctx: context.WithValue(context.Background(), UserContextKey, &GenericPrincipal{
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
				ctx: context.WithValue(context.Background(), UserContextKey, &GenericPrincipal{
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
				ctx: context.WithValue(context.Background(), UserContextKey, NewGenericPrincipalByClaims(jwt.MapClaims{
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
		{
			name: "no user",
			args: args{
				ctx: context.Background(),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want == nil {
				_, ok := GenericIdentityFromContext(tt.args.ctx)
				assert.False(t, ok)
				_, ok = GenericPrincipalFromContext(tt.args.ctx)
				assert.False(t, ok)
				return
			}
			got, _ := GenericIdentityFromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got)
			got2, _ := GenericPrincipalFromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got2.GenericIdentity)
			if got2.Identity().Claims() != nil {
				assert.Equal(t, tt.want.Name(), got2.GenericIdentity.Name())
				assert.Empty(t, got2.GenericIdentity.NameInt())
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
					"sub": "1",
				}}),
				key: "user",
			},
			want:   "1",
			wantOK: true,
		},
		{
			name: "No map claims",
			args: args{
				ctx: context.WithValue(context.Background(), "user", &jwt.Token{Claims: jwt.RegisteredClaims{
					Subject: "1",
				}}),
				key: "user",
			},
			want:   "1",
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := tt.args.ctx.Value(tt.args.key).(*jwt.Token)
			require.True(t, ok)
			v, err := token.Claims.GetSubject()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, v)
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
			got, ok := ctx.Value(UserContextKey).(*GenericPrincipal)
			require.True(t, ok)
			assert.Equal(t, tt.want, *got)
		})
	}
}
