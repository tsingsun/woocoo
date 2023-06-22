package log

import (
	"context"
	"go.uber.org/zap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncomingContext(t *testing.T) {
	type args struct {
		ctx    context.Context
		fields []zap.Field
	}
	tests := []struct {
		name    string
		args    args
		want    *FieldCarrier
		wantErr assert.BoolAssertionFunc
	}{
		{
			name:    "new",
			args:    args{ctx: NewIncomingContext(context.Background(), NewCarrier()), fields: []zap.Field{zap.String("test", "test")}},
			want:    &FieldCarrier{Fields: []zap.Field{zap.String("test", "test")}},
			wantErr: assert.True,
		},
		{
			name:    "context with field",
			args:    args{ctx: NewIncomingContext(context.Background(), NewCarrier(), zap.String("test", "test")), fields: []zap.Field{}},
			want:    &FieldCarrier{Fields: []zap.Field{zap.String("test", "test")}},
			wantErr: assert.True,
		},
		{
			name:    "context without carrier",
			args:    args{ctx: context.Background(), fields: []zap.Field{}},
			want:    nil,
			wantErr: assert.False,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AppendToIncomingContext(tt.args.ctx, tt.args.fields...)
			got, ok := FromIncomingContext(tt.args.ctx)
			if !tt.wantErr(t, ok) {
				return
			}
			assert.EqualValuesf(t, tt.want, got, "AppendToIncomingContext(%v, %v)", tt.args.ctx, tt.args.fields)
		})
	}
}
