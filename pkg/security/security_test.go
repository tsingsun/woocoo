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
				ctx: context.WithValue(context.Background(), PrincipalContextKey, &GenericPrincipal{
					GenericIdentity: &GenericIdentity{
						claims: jwt.MapClaims{
							"sub": "test",
						},
					},
				}),
			},
			want: &GenericIdentity{
				claims: jwt.MapClaims{
					"sub": "test",
				},
			},
		},
		{
			name: "GenericPrincipalFromContext",
			args: args{
				ctx: context.WithValue(context.Background(), PrincipalContextKey, &GenericPrincipal{
					GenericIdentity: &GenericIdentity{
						claims: jwt.MapClaims{
							"sub": "test",
						},
					},
				}),
			},
			want: &GenericIdentity{
				claims: jwt.MapClaims{
					"sub": "test",
				},
			},
		},
		{
			name: "NewGenericPrincipalByClaims",
			args: args{
				ctx: context.WithValue(context.Background(), PrincipalContextKey, NewGenericPrincipalByClaims(jwt.MapClaims{
					"sub": "test",
				})),
			},
			want: &GenericIdentity{
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
				_, ok := FromContext(tt.args.ctx)
				assert.False(t, ok)
				_, ok = FromContext(tt.args.ctx)
				assert.False(t, ok)
				return
			}
			got, _ := FromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got.Identity())
			got2, _ := FromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got2.(*GenericPrincipal).GenericIdentity)
			if got2.Identity().Claims() != nil {
				assert.Equal(t, tt.want.Name(), got2.Identity().Name())
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
			user: GenericPrincipal{GenericIdentity: &GenericIdentity{claims: jwt.MapClaims{
				"sub": "test",
			}}},
			want: GenericPrincipal{GenericIdentity: &GenericIdentity{claims: jwt.MapClaims{
				"sub": "test",
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithContext(ctx, &tt.user)
			got, ok := ctx.Value(PrincipalContextKey).(*GenericPrincipal)
			require.True(t, ok)
			assert.Equal(t, tt.want, *got)
		})
	}
}
