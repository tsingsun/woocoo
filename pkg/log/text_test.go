package log

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
	"time"
)

func TestTextEncoder(t *testing.T) {
	tests := []struct {
		name     string
		input    []any
		do       func(*TextEncoder, ...any)
		expected string
		batch    bool
		noQuotes bool
	}{
		{
			name:  "int",
			input: []any{1, int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1)},
			do: func(enc *TextEncoder, input ...any) {
				funcs := []func(enc *TextEncoder, v any){
					func(enc *TextEncoder, v any) {
						enc.AppendInt(v.(int))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendInt8(v.(int8))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendInt16(v.(int16))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendInt32(v.(int32))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendInt64(v.(int64))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendUint(v.(uint))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendUint8(v.(uint8))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendUint16(v.(uint16))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendUint32(v.(uint32))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendUint64(v.(uint64))
					},
				}
				for i, v := range input {
					enc := enc.cloned()
					funcs[i](enc, v)
					assert.Equal(t, "1", enc.buf.String())
				}
			},
			batch: true,
		},
		{
			name:  "string",
			input: []any{"hello"},
			do: func(enc *TextEncoder, input ...any) {
				enc.AppendString("hello")
			},
			expected: "hello",
		},
		{
			name:  "bytes",
			input: []any{[]byte("hello")},
			do: func(enc *TextEncoder, input ...any) {
				enc.AppendByteString([]byte("hello"))
			},
			expected: "hello",
		},
		{
			name:  "struct",
			input: []any{struct{ A int }{1}},
			do: func(enc *TextEncoder, input ...any) {
				assert.NoError(t, enc.AppendReflected(input[0]))
			},
			expected: "{1}",
		},
		{
			name:  "slice",
			input: []any{[]int{1, 2, 3}},
			do: func(encoder *TextEncoder, input ...any) {
				assert.NoError(t, encoder.AppendReflected(input[0]))
			},
			expected: `"[1 2 3]"`,
			noQuotes: true,
		},
		{
			name:  "array",
			input: []any{[3]int{1, 2, 3}},
			do: func(encoder *TextEncoder, input ...any) {
				assert.NoError(t, encoder.AppendReflected(input[0]))
			},
			expected: `"[1 2 3]"`,
			noQuotes: true,
		},
		{
			name:  "map",
			input: []any{map[string]int{"a": 1, "b": 2}},
			do: func(encoder *TextEncoder, input ...any) {
				assert.NoError(t, encoder.AppendReflected(input[0]))
			},
			expected: `"map[a:1 b:2]"`,
			noQuotes: true,
		},
		{
			name:  "bool",
			input: []any{true},
			do: func(encoder *TextEncoder, input ...any) {
				encoder.AppendBool(input[0].(bool))
			},
			expected: "true",
		},
		{
			name:  "float",
			input: []any{1.1, float32(1.1)},
			do: func(enc *TextEncoder, input ...any) {
				funcs := []func(enc *TextEncoder, v any){
					func(enc *TextEncoder, v any) {
						enc.AppendFloat64(v.(float64))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendFloat32(v.(float32))
					},
				}
				for i, v := range input {
					enc := enc.cloned()
					funcs[i](enc, v)
					assert.Equal(t, "1.1", enc.buf.String())
				}
			},
			batch: true,
		},
		{
			name:  "time",
			input: []any{time.Unix(0, 0).UTC()},
			do: func(encoder *TextEncoder, input ...any) {
				encoder.AppendTime(input[0].(time.Time))
			},
			expected: "1970/01/01 00:00:00.000 +00:00",
		},
		{
			name:  "duration",
			input: []any{time.Second},
			do: func(encoder *TextEncoder, input ...any) {
				encoder.AppendDuration(input[0].(time.Duration))
			},
			expected: "1s",
		},
		{
			name:  "complex",
			input: []any{complex64(complex(1, 2)), complex128(complex(1, 2))},
			do: func(enc *TextEncoder, input ...any) {
				funcs := []func(enc *TextEncoder, v any){
					func(enc *TextEncoder, v any) {
						enc.AppendComplex64(v.(complex64))
					},
					func(enc *TextEncoder, v any) {
						enc.AppendComplex128(v.(complex128))
					},
				}
				for i, v := range input {
					enc := enc.cloned()
					funcs[i](enc, v)
					assert.Equal(t, "1+2i", enc.buf.String())
				}
			},
			batch: true,
		},
		{
			name:  "uintptr",
			input: []any{uintptr(1)},
			do: func(enc *TextEncoder, input ...any) {
				enc.AppendUintptr(input[0].(uintptr))
			},
			expected: "1",
		},
		{
			name:  "error",
			input: []any{errors.New("hello")},
			do: func(enc *TextEncoder, input ...any) {
				assert.NoError(t, enc.AppendReflected(input[0].(error)))
			},
			expected: "hello",
		},
		{
			name: "marshaler",
			input: []any{zapcore.ObjectMarshalerFunc(func(encoder zapcore.ObjectEncoder) error {
				encoder.AddString("foo", "bar")
				encoder.AddInt("baz", 1)
				return nil
			}),
			},
			do: func(enc *TextEncoder, input ...any) {
				assert.NoError(t, enc.AppendObject(input[0].(zapcore.ObjectMarshaler)))
			},
			expected: `"{foo=bar,baz=1}"`,
			noQuotes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewTextEncoder(zap.NewProductionEncoderConfig(), true, true, true)
			if tt.batch {
				tt.do(enc.cloned(), tt.input...)
				return
			}
			tt.do(enc, tt.input[0])
			assert.Equal(t, tt.expected, enc.buf.String())
			if tt.noQuotes {
				enc.Truncate()
				enc.needQuotes = false
				tt.do(enc, tt.input[0])
				assert.Equal(t, tt.expected, fmt.Sprintf("%q", enc.buf.String()))
			}
		})
	}
}
