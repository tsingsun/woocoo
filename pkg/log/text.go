package log

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"math"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// ShortCallerEncoder serializes a caller in file:line format.
func ShortCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(getCallerString(caller))
}

func getCallerString(ec zapcore.EntryCaller) string {
	if !ec.Defined {
		return "<unknown>"
	}

	idx := strings.LastIndexByte(ec.File, '/')
	buf := _pool.Get()
	for i := idx + 1; i < len(ec.File); i++ {
		b := ec.File[i]
		switch {
		case b >= 'A' && b <= 'Z':
			buf.AppendByte(b)
		case b >= 'a' && b <= 'z':
			buf.AppendByte(b)
		case b >= '0' && b <= '9':
			buf.AppendByte(b)
		case b == '.' || b == '-' || b == '_':
			buf.AppendByte(b)
		default:
		}
	}
	buf.AppendByte(':')
	buf.AppendInt(int64(ec.Line))
	caller := buf.String()
	buf.Free()
	return caller
}

// For JSON-escaping; see TextEncoder.safeAddString below.
const _hex = "0123456789abcdef"

var _textPool = sync.Pool{New: func() any {
	return &TextEncoder{}
}}

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
)

func getTextEncoder() *TextEncoder {
	return _textPool.Get().(*TextEncoder)
}

func putTextEncoder(enc *TextEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_textPool.Put(enc)
}

var _ zapcore.Encoder = (*TextEncoder)(nil)
var _ zapcore.ArrayEncoder = (*TextEncoder)(nil)

type TextEncoder struct {
	*zapcore.EncoderConfig
	buf                 *buffer.Buffer
	spaced              bool // include spaces after colons and commas
	openNamespaces      int
	disableErrorVerbose bool
	needQuotes          bool

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder
}

// NewTextEncoder creates a fast, low-allocation Text encoder. The encoder
// appropriately escapes all field keys and values.
// log format see https://github.com/tikv/rfcs/blob/master/text/0018-unified-log-format.md#log-fields-section.
// in same scenarios, you can use needQuotes=false to remove quotes if no need followed log-format.
func NewTextEncoder(config zapcore.EncoderConfig, disableErrorVerbose, disableTimestamp, needQuotes bool) *TextEncoder {
	cc := zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:        config.TimeKey,
		LevelKey:       config.LevelKey,
		NameKey:        config.NameKey,
		CallerKey:      config.CallerKey,
		MessageKey:     config.MessageKey,
		StacktraceKey:  config.StacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     DefaultTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   ShortCallerEncoder,
	}
	if disableTimestamp {
		cc.TimeKey = ""
	}
	return &TextEncoder{
		EncoderConfig:       &cc,
		buf:                 _pool.Get(),
		spaced:              false,
		disableErrorVerbose: disableErrorVerbose,
		needQuotes:          needQuotes,
	}
}

// buildTextEncoder builds a TextEncoder with the given config.
func buildTextEncoder(cfg *Config) zapcore.Encoder {
	cc := zap.NewProductionEncoderConfig()
	if len(cfg.ZapConfigs) > 0 {
		cc = cfg.ZapConfigs[0].EncoderConfig
	}
	return NewTextEncoder(cc, cfg.DisableErrorVerbose, cfg.DisableTimestamp, true)
}

func (enc *TextEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *TextEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *TextEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *TextEncoder) AddByteString(key string, val []byte) {
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *TextEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *TextEncoder) AddComplex128(key string, val complex128) {
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *TextEncoder) AddDuration(key string, val time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *TextEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *TextEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.AppendInt64(val)
}

func (enc *TextEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = _pool.Get()
		enc.reflectEnc = json.NewEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

func (enc *TextEncoder) AddReflected(key string, obj any) error {
	enc.resetReflectBuf()
	err := enc.reflectEnc.Encode(obj)
	if err != nil {
		return err
	}
	enc.reflectBuf.TrimNewline()
	enc.addKey(key)
	enc.AppendByteString(enc.reflectBuf.Bytes())
	return nil
}

func (enc *TextEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

func (enc *TextEncoder) AddString(key, val string) {
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *TextEncoder) AddTime(key string, val time.Time) {
	enc.addKey(key)
	enc.AppendTime(val)
}

func (enc *TextEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *TextEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	enc.addElementSeparator()
	ne := enc.cloned()
	ne.buf.AppendByte('[')
	err := arr.MarshalLogArray(ne)
	ne.buf.AppendByte(']')
	enc.AppendByteString(ne.buf.Bytes())
	ne.buf.Free()
	putTextEncoder(ne)
	return err
}

func (enc *TextEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	enc.addElementSeparator()
	ne := enc.cloned()
	ne.buf.AppendByte('{')
	err := obj.MarshalLogObject(ne)
	ne.buf.AppendByte('}')
	enc.AppendByteString(ne.buf.Bytes())
	ne.buf.Free()
	putTextEncoder(ne)
	return err
}

func (enc *TextEncoder) AppendBool(val bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *TextEncoder) AppendByteString(val []byte) {
	enc.addElementSeparator()
	if !enc.needDoubleQuotes(string(val)) {
		enc.safeAddByteString(val)
		return
	}
	enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	enc.buf.AppendByte('"')
}

func (enc *TextEncoder) AppendComplex128(val complex128) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := real(val), imag(val)
	enc.buf.AppendFloat(r, 64)
	enc.buf.AppendByte('+')
	enc.buf.AppendFloat(i, 64)
	enc.buf.AppendByte('i')
}

func (enc *TextEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	enc.EncodeDuration(val, enc)
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *TextEncoder) AppendInt64(val int64) {
	enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *TextEncoder) AppendReflected(val any) error {
	enc.AppendString(fmt.Sprintf("%v", val))
	return nil
}

func (enc *TextEncoder) AppendString(val string) {
	enc.addElementSeparator()
	enc.safeAddStringWithQuote(val)
}

func (enc *TextEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	enc.EncodeTime(val, enc)
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *TextEncoder) beginQuoteFiled() {
	if enc.buf.Len() > 0 {
		enc.buf.AppendByte(' ')
	}
	enc.buf.AppendByte('[')
}

func (enc *TextEncoder) endQuoteFiled() {
	enc.buf.AppendByte(']')
}

func (enc *TextEncoder) AppendUint64(val uint64) {
	enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *TextEncoder) AddComplex64(k string, v complex64) { enc.AddComplex128(k, complex128(v)) }
func (enc *TextEncoder) AddFloat32(k string, v float32)     { enc.AddFloat64(k, float64(v)) }
func (enc *TextEncoder) AddInt(k string, v int)             { enc.AddInt64(k, int64(v)) }
func (enc *TextEncoder) AddInt32(k string, v int32)         { enc.AddInt64(k, int64(v)) }
func (enc *TextEncoder) AddInt16(k string, v int16)         { enc.AddInt64(k, int64(v)) }
func (enc *TextEncoder) AddInt8(k string, v int8)           { enc.AddInt64(k, int64(v)) }
func (enc *TextEncoder) AddUint(k string, v uint)           { enc.AddUint64(k, uint64(v)) }
func (enc *TextEncoder) AddUint32(k string, v uint32)       { enc.AddUint64(k, uint64(v)) }
func (enc *TextEncoder) AddUint16(k string, v uint16)       { enc.AddUint64(k, uint64(v)) }
func (enc *TextEncoder) AddUint8(k string, v uint8)         { enc.AddUint64(k, uint64(v)) }
func (enc *TextEncoder) AddUintptr(k string, v uintptr)     { enc.AddUint64(k, uint64(v)) }
func (enc *TextEncoder) AppendComplex64(v complex64)        { enc.AppendComplex128(complex128(v)) }
func (enc *TextEncoder) AppendFloat64(v float64)            { enc.appendFloat(v, 64) }
func (enc *TextEncoder) AppendFloat32(v float32)            { enc.appendFloat(float64(v), 32) }
func (enc *TextEncoder) AppendInt(v int)                    { enc.AppendInt64(int64(v)) }
func (enc *TextEncoder) AppendInt32(v int32)                { enc.AppendInt64(int64(v)) }
func (enc *TextEncoder) AppendInt16(v int16)                { enc.AppendInt64(int64(v)) }
func (enc *TextEncoder) AppendInt8(v int8)                  { enc.AppendInt64(int64(v)) }
func (enc *TextEncoder) AppendUint(v uint)                  { enc.AppendUint64(uint64(v)) }
func (enc *TextEncoder) AppendUint32(v uint32)              { enc.AppendUint64(uint64(v)) }
func (enc *TextEncoder) AppendUint16(v uint16)              { enc.AppendUint64(uint64(v)) }
func (enc *TextEncoder) AppendUint8(v uint8)                { enc.AppendUint64(uint64(v)) }
func (enc *TextEncoder) AppendUintptr(v uintptr)            { enc.AppendUint64(uint64(v)) }

func (enc *TextEncoder) Truncate() {
	enc.buf.Reset()
}

func (enc *TextEncoder) Clone() zapcore.Encoder {
	clone := enc.cloned()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *TextEncoder) cloned() *TextEncoder {
	clone := getTextEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.disableErrorVerbose = enc.disableErrorVerbose
	clone.needQuotes = enc.needQuotes
	clone.buf = _pool.Get()
	return clone
}

func (enc *TextEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.cloned()
	if final.TimeKey != "" {
		final.beginQuoteFiled()
		final.AppendTime(ent.Time)
		final.endQuoteFiled()
	}

	if final.LevelKey != "" {
		final.beginQuoteFiled()
		cur := final.buf.Len()
		final.EncodeLevel(ent.Level, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeLevel was a no-op. Fall back to string to keep
			// output JSON valid.
			final.AppendString(ent.Level.String())
		}
		final.endQuoteFiled()
	}

	if ent.LoggerName != "" && final.NameKey != "" {
		final.beginQuoteFiled()
		cur := final.buf.Len()
		nameEncoder := final.EncodeName

		// if no name encoder provided, fall back to FullNameEncoder for backwards
		// compatibility
		if nameEncoder == nil {
			nameEncoder = zapcore.FullNameEncoder
		}

		nameEncoder(ent.LoggerName, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeName was a no-op. Fall back to string to
			// keep output JSON valid.
			final.AppendString(ent.LoggerName)
		}
		final.endQuoteFiled()
	}
	if ent.Caller.Defined && final.CallerKey != "" {
		final.beginQuoteFiled()
		cur := final.buf.Len()
		final.EncodeCaller(ent.Caller, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeCaller was a no-op. Fall back to string to
			// keep output JSON valid.
			final.AppendString(ent.Caller.String())
		}
		final.endQuoteFiled()
	}
	// add Message
	if len(ent.Message) > 0 {
		final.beginQuoteFiled()
		final.AppendString(ent.Message)
		final.endQuoteFiled()
	}
	if enc.buf.Len() > 0 {
		final.buf.AppendByte(' ')
		final.buf.Write(enc.buf.Bytes())
	}
	final.addFields(fields)
	final.closeOpenNamespaces()
	if ent.Stack != "" && final.StacktraceKey != "" {
		final.beginQuoteFiled()
		final.AddString(final.StacktraceKey, ent.Stack)
		final.endQuoteFiled()
	}

	if final.LineEnding != "" {
		final.buf.AppendString(final.LineEnding)
	} else {
		final.buf.AppendString(zapcore.DefaultLineEnding)
	}

	ret := final.buf
	putTextEncoder(final)
	return ret, nil
}

func (enc *TextEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
}

func (enc *TextEncoder) addKey(key string) {
	enc.addElementSeparator()
	enc.safeAddStringWithQuote(key)
	enc.buf.AppendByte('=')
}

func (enc *TextEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ', '=':
		return
	default:
		enc.buf.AppendByte(',')
	}
}

func (enc *TextEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString("NaN")
	case math.IsInf(val, 1):
		enc.buf.AppendString("+Inf")
	case math.IsInf(val, -1):
		enc.buf.AppendString("-Inf")
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *TextEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddStringWithQuote will automatically add quotoes.
func (enc *TextEncoder) safeAddStringWithQuote(s string) {
	if !enc.needDoubleQuotes(s) {
		enc.safeAddString(s)
		return
	}
	enc.buf.AppendByte('"')
	enc.safeAddString(s)
	enc.buf.AppendByte('"')
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

// See [log-fileds](https://github.com/tikv/rfcs/blob/master/text/0018-unified-log-format.md#log-fields-section).
func (enc *TextEncoder) needDoubleQuotes(s string) bool {
	if !enc.needQuotes {
		return false
	}
	for i := 0; i < len(s); {
		b := s[i]
		if b <= 0x20 {
			return true
		}
		switch b {
		case '\\', '"', '[', ']', '=':
			return true
		}
		i++
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

func (enc *TextEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func (enc *TextEncoder) addFields(fields []zapcore.Field) {
	for _, f := range fields {
		if f.Type == zapcore.ErrorType {
			enc.encodeError(f)
			continue
		}
		enc.beginQuoteFiled()
		f.AddTo(enc)
		enc.endQuoteFiled()
	}
}

func (enc *TextEncoder) encodeError(f zapcore.Field) {
	err := f.Interface.(error)
	basic := err.Error()
	enc.beginQuoteFiled()
	enc.AddString(f.Key, basic)
	enc.endQuoteFiled()
	if enc.disableErrorVerbose {
		return
	}
	if e, isFormatter := err.(fmt.Formatter); isFormatter {
		verbose := fmt.Sprintf("%+v", e)
		if verbose != basic {
			// This is a rich error type, like those produced by
			// github.com/pkg/errors.
			enc.beginQuoteFiled()
			enc.AddString(f.Key+"Verbose", verbose)
			enc.endQuoteFiled()
		}
	}
}

// String returns the encoded log message.
func (enc *TextEncoder) String() string {
	return enc.buf.String()
}
