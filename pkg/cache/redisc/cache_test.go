package redisc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSkipMode(t *testing.T) {
	type args struct {
		mode SkipMode
	}
	tests := []struct {
		name string
		f    SkipMode
		Func func(mode SkipMode) bool
		args args
		want bool
	}{
		{
			name: "in",
			f:    SkipLocal,
			Func: SkipLocal.Is,
			args: args{
				mode: SkipRedis,
			},
			want: false,
		},
		{
			name: "any",
			f:    SkipLocal,
			Func: SkipLocal.Is,
			args: args{
				mode: SkipMode(0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.Func(tt.args.mode), "Is(%v)", tt.args.mode)
		})
	}
}
