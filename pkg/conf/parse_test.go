package conf

import (
	"bytes"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/tsingsun/woocoo/test/testdata"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
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
			wantErr: assert.Error,
		},
		{
			name: "simple",
			args: args{buf: func() io.Reader {
				return bytes.NewBuffer([]byte("a:\n -b\n -c"))
			}()},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewParserFromBuffer(tt.args.buf)
			if !tt.wantErr(t, err) {
				return
			}
			if got != nil {
				assert.NotNil(t, got.Operator())
			}
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
			fields: fields{parser: NewParserFromStringMap(
				map[string]any{"map": map[string]any{
					"a": map[string]any{"a": 1},
					"b": nil,
				}})},
			args: args{
				key: "map",
				dst: make(map[string]*parseTarget),
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Equal(t, map[string]*parseTarget{
					"a": {A: 1}, "b": {},
				}, i[1])
				return false
			},
		},
		{
			name: "map exist key will new",
			fields: fields{parser: NewParserFromStringMap(
				map[string]any{"map": map[string]any{
					"a": map[string]any{"a": 1},
					"b": nil,
				}})},
			args: args{
				key: "map",
				dst: map[string]*parseTarget{
					"a": {B: "s"}, "b": {},
					"c": {},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Equal(t, map[string]*parseTarget{
					"a": {A: 1}, "b": {}, "c": {},
				}, i[1])
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
			name:   "slice not merge",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args: args{
				key: "slice",
				dst: []string{"c", "d", "e", "f"},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Len(t, i[1].([]string), 3)
				assert.Equal(t, []string{"a", "b", "c"}, i[1])
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
			name:   "keepStructData",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"struct": map[string]any{"a": 1, "c": []string{"c1", "c2"}}})},
			args: args{
				key: "struct",
				dst: parseTarget{
					A: 2,
					B: "string",
					C: []string{"c3", "c4", "c5"},
				},
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
			name: "sliceStruct-all",
			fields: fields{parser: NewParserFromStringMap(
				map[string]any{"structAll": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
			args: args{
				key: "",
				dst: struct {
					StructAll *parseTarget
				}{},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.EqualValues(t, struct {
					StructAll *parseTarget
				}{
					StructAll: &parseTarget{1, "string", []string{"c1", "c2"}},
				}, i[1])
				return false
			},
		},
		{
			name: "pointer",
			fields: fields{parser: NewParserFromStringMap(
				map[string]any{"struct": map[string]any{"a": 1, "b": "string", "c": []string{"c1", "c2"}}})},
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
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
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
			name:   "slice no merge",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args: args{
				key: "slice",
				dst: []string{"c", "d", "e", "f"},
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.NoError(t, err)
				assert.Len(t, i[1].([]string), 3)
				assert.Equal(t, []string{"a", "b", "c"}, i[1])
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

func TestParser_MergeStringMap(t *testing.T) {
	type fields struct {
		parser *Parser
	}
	type args struct {
		cfg map[string]any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
		check   func(*Parser, map[string]any)
	}{
		{
			name:   "sliceString",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args: args{
				cfg: map[string]any{"slice": []string{"d", "e", "f"}},
			},
			wantErr: assert.NoError,
			check: func(l *Parser, ori map[string]any) {
				assert.EqualValues(t, []string{"d", "e", "f"}, l.Get("slice").([]string))
			},
		},
		{
			name:   "mapString",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"k1": "a"})},
			args: args{
				cfg: map[string]any{"k1": "b"},
			},
			wantErr: assert.NoError,
			check: func(l *Parser, ori map[string]any) {
				assert.EqualValues(t, "b", l.Get("k1").(string))
				assert.EqualValues(t, "b", ori["k1"].(string))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.fields.parser
			err := l.MergeStringMap(tt.args.cfg)
			tt.wantErr(t, err)
			tt.check(l, tt.args.cfg)
		})
	}
}

func TestNewParserFromFile(t *testing.T) {
	t.Run("app", func(t *testing.T) {
		l, err := NewParserFromFile(testdata.Path("etc/app.yaml"))
		assert.NoError(t, err)
		bs, err := l.ToBytes(yaml.Parser())
		assert.NoError(t, err)
		assert.NotNil(t, bs)
	})
}
