package conf

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

type (
	parseTarget struct {
		A int
		B string
		C []string
	}
	parseTarget2 struct {
		text string
	}
	parseInline struct {
		InLineTarget
	}
	InLineTarget struct {
		Inline int
	}
)

func (p *parseTarget2) UnmarshalText(text []byte) error {
	p.text = string(text)
	return nil
}

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
			name: "map",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"map": map[string]any{
				"a": map[string]any{"a": 1}}})},
			args: args{
				key: "map",
				dst: make(map[string]*parseTarget),
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Equal(t, map[string]*parseTarget{"a": {A: 1}}, i[1])
				return false
			},
		},
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
				dst: parseTarget{},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.EqualValues(t, parseTarget{
					1, "string", []string{"c1", "c2"},
				}, i[1])
				return false
			},
		},
		{
			name:   "sliceStruct-all",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"structAll": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
			args: args{
				key: "",
				dst: struct {
					StructAll parseTarget
				}{},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.EqualValues(t, struct {
					StructAll parseTarget
				}{
					StructAll: parseTarget{1, "string", []string{"c1", "c2"}},
				}, i[1])
				return false
			},
		},
		{
			name:   "pointer",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"struct": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
			args: args{
				key: "struct",
				dst: new(parseTarget),
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Equal(t, &parseTarget{
					1, "string", []string{"c1", "c2"},
				}, i[1])
				return false
			},
		},
		{
			name:   "textUnmarshaler",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"struct": "text"})},
			args: args{
				key: "struct",
				dst: (*parseTarget2)(nil),
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				tg := &parseTarget2{
					text: "text",
				}
				assert.NoError(t, err)
				assert.Equal(t, tg, i[1])
				return false
			},
		},
		{
			name: "squash",
			fields: fields{parser: NewParserFromStringMap(map[string]any{
				"struct": map[string]any{
					"target": map[string]any{"inline": 1},
				}}),
			},
			args: args{
				key: "struct.target",
				dst: parseInline{},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				assert.Equal(t, parseInline{InLineTarget{Inline: 1}}, i[1])
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

func TestParser_UnmarshalExact(t *testing.T) {
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
		{
			name: "sliceStruct-empty-key",
			fields: fields{parser: NewParserFromStringMap(map[string]any{
				"struct": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
			args: args{
				key: "",
				dst: struct {
					A int
					B string
					C []string
				}{},
			},
			wantErr: assert.Error,
		},
		{
			name: "sliceStruct-err",
			fields: fields{parser: NewParserFromStringMap(map[string]any{
				"struct": map[string]any{
					"a": 1, "b": "string",
					"c": []string{"c1", "c2"},
					"d": "string",
				}})},
			args: args{
				key: "struct",
				dst: struct {
					A int
					B string
					C []string
				}{},
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.fields.parser
			tt.wantErr(t, l.UnmarshalExact(tt.args.key, &tt.args.dst), tt.args.key, tt.args.dst)
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
