package log

import (
	"encoding/base64"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"golang.org/x/text/encoding"
	"math"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	_hex = "0123456789abcdef"
)

type (
	TextEncoder struct {
		*zapcore.EncoderConfig
		buf    *buffer.Buffer
		spaced bool

		// for encoding generic values by reflection
		reflectBuf *buffer.Buffer
		reflectEnc *encoding.Encoder
	}
)

var (
	textpool = sync.Pool{New: func() interface{} {
		return &TextEncoder{}
	}}
	buffpoll = buffer.NewPool()
)

var _ zapcore.Encoder = (*TextEncoder)(nil)
var _ zapcore.ArrayEncoder = (*TextEncoder)(nil)

func NewTextEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &TextEncoder{EncoderConfig: &cfg, buf: buffpoll.Get()}
}

func (enc *TextEncoder) addKey(key string) {
	enc.addElementSeparator()
	if enc.spaced {
		enc.buf.AppendByte(' ')
	}
}

func (enc *TextEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		enc.buf.AppendByte(',')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *TextEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *TextEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *TextEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *TextEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// Clone copies the encoder, ensuring that adding fields to the copy doesn't
// affect the original.
func (enc *TextEncoder) Clone() zapcore.Encoder { return enc }

// EncodeEntry encodes an entry and fields, along with any accumulated
// context, into a byte buffer and returns ienc. Any fields that are empty,
// including fields on the `Entry` type, should be omitted.
func (enc *TextEncoder) EncodeEntry(zapcore.Entry, []zapcore.Field) (buf *buffer.Buffer, err error) {
	return
}

// Logging-specific marshalers.
func (enc *TextEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) (err error)   { return }
func (enc *TextEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) (err error) { return }

func (enc *TextEncoder) AddComplex64(key string, value complex64) {
	enc.AddComplex128(key, complex128(value))
}
func (enc *TextEncoder) AddFloat32(key string, value float32) {
	enc.AddFloat64(key, float64(value))
}
func (enc *TextEncoder) AddInt(key string, value int) {
	enc.AddInt64(key, int64(value))
}
func (enc *TextEncoder) AddInt32(key string, value int32) {
	enc.AddInt64(key, int64(value))
}
func (enc *TextEncoder) AddInt16(key string, value int16) {
	enc.AddInt64(key, int64(value))
}
func (enc *TextEncoder) AddInt8(key string, value int8) {
	enc.AddInt64(key, int64(value))
}
func (enc *TextEncoder) AddUint32(key string, value uint32) {
	enc.AddUint64(key, uint64(value))
}
func (enc *TextEncoder) AddUint(key string, value uint) {
	enc.AddUint64(key, uint64(value))
}
func (enc *TextEncoder) AddUint16(key string, value uint16) {
	enc.AddUint64(key, uint64(value))
}
func (enc *TextEncoder) AddUint8(key string, value uint8) {
	enc.AddUint64(key, uint64(value))
}
func (enc *TextEncoder) AddUintptr(key string, value uintptr) {
	enc.AddUint64(key, uint64(value))
}

func (enc *TextEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = buffpoll.Get()
		enc.reflectEnc = &encoding.Encoder{}

		// For consistency with our custom JSON encoder.
		// enc.reflectEnc.SetEscapeHTML(false)
	} else {
		enc.reflectBuf.Reset()
	}
}

var nullLiteralBytes = []byte("null")

func (enc *TextEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	return nil, nil
}

// AddReflected uses reflection to serialize arbitrary objects, so it can be
// slow and allocation-heavy.
func (enc *TextEncoder) AddReflected(key string, value interface{}) (err error) {
	var valueBytes []byte
	valueBytes, err = enc.encodeReflected(value)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return
}

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (enc *TextEncoder) OpenNamespace(key string) {}

// Built-in types.
// for arbitrary bytes
func (enc *TextEncoder) AddBinary(key string, value []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(value))
}
func (enc *TextEncoder) AddDuration(key string, value time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(value, enc)
	}
	if cur == enc.buf.Len() {
		enc.AppendInt64(int64(value))
	}
}
func (enc *TextEncoder) AddComplex128(key string, value complex128) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(value)), float64(imag(value))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, 64)
	enc.buf.AppendByte('+')
	enc.buf.AppendFloat(i, 64)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}
func (enc *TextEncoder) AddByteString(key string, value []byte) {
	enc.addKey(key)
	enc.AppendByteString(value)
}
func (enc *TextEncoder) AddFloat64(key string, value float64) {
	enc.addKey(key)
	enc.appendFloat(value, 64)
}
func (enc *TextEncoder) AddTime(key string, value time.Time) {
	enc.addKey(key)
	enc.buf.AppendTime(value, time.RFC3339)
}
func (enc *TextEncoder) AddUint64(key string, value uint64) {
	enc.addKey(key)
	enc.buf.AppendUint(value)
}
func (enc *TextEncoder) AddInt64(key string, value int64) {
	enc.addKey(key)
	enc.buf.AppendInt(value)
}
func (enc *TextEncoder) AddBool(key string, value bool) {
	enc.addKey(key)
	enc.buf.AppendBool(value)
}
func (enc *TextEncoder) AddString(key, value string) {
	enc.addKey(key)
	enc.buf.AppendString(value)
}

// ArrayEncoder

// Time-related types.
func (enc *TextEncoder) AppendDuration(value time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(value, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op.
		enc.AppendString(value.String())
	}
}

func (enc *TextEncoder) AppendTime(value time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(value, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op.
		enc.AppendString(value.Format(time.RFC3339))
	}
}

// Logging-specific marshalers.{}
func (enc *TextEncoder) AppendArray(arr zapcore.ArrayMarshaler) (err error) {
	enc.addElementSeparator()
	err = arr.MarshalLogArray(enc)
	return
}

func (enc *TextEncoder) AppendObject(obj zapcore.ObjectMarshaler) (err error) {
	enc.addElementSeparator()
	err = obj.MarshalLogObject(enc)
	return
}

// AppendReflected uses reflection to serialize arbitrary objects, so it's{}
// slow and allocation-heavy.{}
func (enc *TextEncoder) AppendReflected(value interface{}) (err error) {
	// TODO
	return
}

func (enc *TextEncoder) AppendBool(value bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(value)
}

// for UTF-8 encoded bytes
func (enc *TextEncoder) AppendByteString(value []byte) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddByteString(value)
	enc.buf.AppendByte('"')
}

func (enc *TextEncoder) AppendComplex128(value complex128) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(value)), float64(imag(value))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, 64)
	enc.buf.AppendByte('+')
	enc.buf.AppendFloat(i, 64)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *TextEncoder) AppendUint64(value uint64)       { enc.buf.AppendUint(value) }
func (enc *TextEncoder) AppendString(value string)       { enc.buf.AppendString(value) }
func (enc *TextEncoder) AppendInt64(value int64)         { enc.buf.AppendInt(value) }
func (enc *TextEncoder) AppendFloat64(value float64)     { enc.appendFloat(value, 64) }
func (enc *TextEncoder) AppendComplex64(value complex64) { enc.AppendComplex128(complex128(value)) }
func (enc *TextEncoder) AppendFloat32(value float32)     { enc.AppendFloat64(float64(value)) }
func (enc *TextEncoder) AppendInt32(value int32)         { enc.AppendInt64(int64(value)) }
func (enc *TextEncoder) AppendInt16(value int16)         { enc.AppendInt64(int64(value)) }
func (enc *TextEncoder) AppendInt(value int)             { enc.AppendInt64(int64(value)) }
func (enc *TextEncoder) AppendInt8(value int8)           { enc.AppendInt64(int64(value)) }
func (enc *TextEncoder) AppendUint(value uint)           { enc.AppendUint64(uint64(value)) }
func (enc *TextEncoder) AppendUint32(value uint32)       { enc.AppendUint64(uint64(value)) }
func (enc *TextEncoder) AppendUint16(value uint16)       { enc.AppendUint64(uint64(value)) }
func (enc *TextEncoder) AppendUint8(value uint8)         { enc.AppendUint64(uint64(value)) }
func (enc *TextEncoder) AppendUintptr(value uintptr)     { enc.AppendUint64(uint64(value)) }
