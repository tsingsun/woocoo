package security

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
				ctx: context.WithValue(context.Background(), "user", &jwt.Token{Claims: jwt.MapClaims{
					"NoExist": 1,
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
