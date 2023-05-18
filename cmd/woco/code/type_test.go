package code

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestParseGoType(t *testing.T) {
	type aliasUUID uuid.UUID
	type args struct {
		typ any
	}
	tests := []struct {
		name    string
		args    args
		want    *RType
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "string",
			args: args{
				typ: "",
			},
			want: &RType{
				Name:    "string",
				Kind:    reflect.String,
				Ident:   "string",
				PkgPath: "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "string slice",
			args: args{
				typ: []string{},
			},
			want: &RType{
				Name:    "",
				Kind:    reflect.Slice,
				Ident:   "[]string",
				PkgPath: "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "string pointer",
			args: args{
				typ: new(string),
			},
			want: &RType{
				Name:    "string",
				Kind:    reflect.Ptr,
				Ident:   "string",
				PkgPath: "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "string map",
			args: args{
				typ: make(map[string]string),
			},
			want: &RType{
				Name:    "",
				Kind:    reflect.Map,
				Ident:   "map[string]string",
				PkgPath: "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "uuid",
			args: args{
				typ: uuid.New(),
			},
			want: &RType{
				Name:    "UUID",
				Ident:   "uuid.UUID",
				Kind:    reflect.Array,
				PkgPath: "github.com/google/uuid",
			},
			wantErr: assert.NoError,
		},
		{
			name: "uuid slice",
			args: args{
				typ: []*uuid.UUID{new(uuid.UUID)},
			},
			want: &RType{
				Name:    "",
				Ident:   "[]*uuid.UUID",
				Kind:    reflect.Slice,
				PkgPath: "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "alias uuid",
			args: args{
				typ: new(aliasUUID),
			},
			want: &RType{
				Name:    "aliasUUID",
				Ident:   "code.aliasUUID",
				Kind:    reflect.Pointer,
				PkgPath: "github.com/tsingsun/woocoo/cmd/woco/code",
				Methods: nil, //empty
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGoType(tt.args.typ)
			if !tt.wantErr(t, err, fmt.Sprintf("ParseGoType(%v)", tt.args.typ)) {
				return
			}
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Kind, got.Kind)
			assert.Equal(t, tt.want.Ident, got.Ident)
			assert.Equal(t, tt.want.PkgPath, got.PkgPath)
		})
	}
}
