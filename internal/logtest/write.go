package logtest

import (
	"bytes"
	"errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"strings"
)

// A Syncer is a spy for the Sync portion of zapcore.WriteSyncer.
type Syncer struct {
	err    error
	called bool
}

// SetError sets the error that the Sync method will return.
func (s *Syncer) SetError(err error) {
	s.err = err
}

// Sync records that it was called, then returns the user-supplied error (if
// any).
func (s *Syncer) Sync() error {
	s.called = true
	return s.err
}

// Called reports whether the Sync method was called.
func (s *Syncer) Called() bool {
	return s.called
}

// A Discarder sends all writes to io.Discard.
type Discarder struct{ Syncer }

// Write implements io.Writer.
func (d *Discarder) Write(b []byte) (int, error) {
	return io.Discard.Write(b)
}

// FailWriter is a WriteSyncer that always returns an error on writes.
type FailWriter struct{ Syncer }

// Write implements io.Writer.
func (w FailWriter) Write(b []byte) (int, error) {
	return len(b), errors.New("failed")
}

// ShortWriter is a WriteSyncer whose write method never fails, but
// nevertheless fails to the last byte of the input.
type ShortWriter struct{ Syncer }

// Write implements io.Writer.
func (w ShortWriter) Write(b []byte) (int, error) {
	return len(b) - 1, nil
}

// Buffer is an implementation of zapcore.WriteSyncer that sends all writes to
// a bytes.Buffer. It has convenience methods to split the accumulated buffer
// on newlines.
type Buffer struct {
	bytes.Buffer
	Syncer
}

// Lines returns the current buffer contents, split on newlines.
func (b *Buffer) Lines() []string {
	output := strings.Split(b.String(), "\n")
	return output[:len(output)-1]
}

// Stripped returns the current buffer contents with the last trailing newline
// stripped.
func (b *Buffer) Stripped() string {
	return strings.TrimRight(b.String(), "\n")
}

func (b *Buffer) LastLine() string {
	l := b.Lines()
	return l[len(l)-1]
}

func NewBuffCore(ws *Buffer) zapcore.Core {
	en := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	return zapcore.NewCore(en, ws, zap.DebugLevel)
}

func NewBuffLogger(ws *Buffer, opts ...zap.Option) *zap.Logger {
	// production config contains stacktrace setting
	std, err := zap.NewProduction(opts...)
	if err != nil {
		panic(err)
	}
	core := zapcore.NewTee(std.Core(), NewBuffCore(ws))
	return zap.New(core, opts...)
}
