package conf

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestNewParserFromBuffer(t *testing.T) {
	type args struct {
		buf io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *Parser
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "sliceRootNoSupport",
			args: args{buf: func() io.Reader {
				return bytes.NewBuffer([]byte("-a\n-b\n-c"))
			}()},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewParserFromBuffer(tt.args.buf)
			if !tt.wantErr(t, err, fmt.Sprintf("NewParserFromBuffer(%v)", tt.args.buf)) {
				return
			}
			assert.Equalf(t, tt.want, got, "NewParserFromBuffer(%v)", tt.args.buf)
		})
	}
}

func TestParser_Unmarshal(t *testing.T) {
	type fields struct {
		parser *Parser
	}
	type args struct {
		key string
		dst any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "sliceString",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args: args{
				key: "slice",
				dst: []string{},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Len(t, i[1].([]string), 3)
				return false
			},
		},
		{
			name:   "sliceMerge",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args: args{
				key: "slice",
				dst: []string{"c", "d", "e", "f"},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Len(t, i[1].([]string), 4)
				assert.Equal(t, []string{"a", "b", "c", "f"}, i[1])
				return false
			},
		},
		{
			name:   "sliceStruct",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"struct": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
			args: args{
				key: "struct",
				dst: struct {
					A int
					B string
					C []string
				}{},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.EqualValues(t, struct {
					A int
					B string
					C []string
				}{
					1, "string", []string{"c1", "c2"},
				}, i[1])
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.fields.parser
			tt.wantErr(t, l.Unmarshal(tt.args.key, &tt.args.dst), tt.args.key, tt.args.dst)
		})
	}
}

func TestParser_Sub(t *testing.T) {
	type fields struct {
		parser *Parser
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "sliceString",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args: args{
				key: "slice",
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.Error(t, err)
				assert.Nil(t, i[0])
				return false
			},
		},
		{
			name: "sliceString",
			fields: fields{parser: func() *Parser {
				str := "slice:\n"
				p, err := NewParserFromBuffer(bytes.NewReader([]byte(str)))
				if assert.NoError(t, err) {
					return p
				}
				return nil
			}()},
			args: args{
				key: "slice",
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.NotNil(t, i[0])
				return false
			},
		},
		{
			name:   "sliceStruct",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"struct": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
			args: args{
				key: "struct",
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.NotNil(t, i[0])
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.fields.parser
			got, err := l.Sub(tt.args.key)
			tt.wantErr(t, err, got)
		})
	}
}
