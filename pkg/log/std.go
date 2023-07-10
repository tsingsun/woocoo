package log

import (
	"bytes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

var logo = `
 ___      _______________________________ 
__ | /| / /  __ \  __ \  ___/  __ \  __ \
__ |/ |/ // /_/ / /_/ / /__ / /_/ / /_/ /
____/|__/ \____/\____/\___/ \____/\____/
`

// Println wrapper native log.Println
func Println(v ...any) {
	log.Println(v...)
}

// Printf wrapper native log.Printf
func Printf(format string, v ...any) {
	log.Printf(format, v...)
}

func PrintLogo() {
	Println(logo)
}

// Writer is an io.Writer that writes to the provided Zap logger. It is inspirited zapio.Writer
type Writer struct {
	Log *zap.Logger

	// Default log level for the messages which can't inspect a level.
	//
	// If unspecified, defaults to Info.
	Level zapcore.Level

	buff bytes.Buffer
}

func (w *Writer) Check(bs []byte) (lvl zapcore.Level, miss bool, msgIndex int) {
	lvl = w.Level
	x := bytes.IndexByte(bs, '[')
	if x >= 0 {
		y := bytes.IndexByte(bs[x:], ']')
		if y >= 0 {
			if lvl.Set(string(bs[x+1:x+y])) != nil {
				miss = true
			}
			msgIndex = x + y + 1
		}
	}
	return
}

// Write writes the provided bytes to the underlying logger at the configured
// log level and returns the length of the bytes.
//
// Write will split the input on newlines and post each line as a new log entry
// to the logger.
func (w *Writer) Write(bs []byte) (n int, err error) {
	lvl, miss, idx := w.Check(bs)
	// Skip all checks if the level isn't enabled.
	if !miss && !w.Log.Core().Enabled(lvl) {
		return len(bs), nil
	}

	n = len(bs)
	for len(bs) > 0 {
		bs = w.writeLine(bs, lvl, idx)
	}

	return n, nil
}

// writeLine writes a single line from the input, returning the remaining,
// unconsumed bytes.
func (w *Writer) writeLine(line []byte, lvl zapcore.Level, bodyidx int) (remaining []byte) {
	idx := bytes.IndexByte(line, '\n')
	if idx < 0 {
		w.buff.Write(line)
		// If there are no newlines, buffer the entire string.
		return nil
	}

	// Split on the newline, buffer and flush the left.
	line, remaining = line[:idx], line[idx+1:]
	// Fast path: if we don't have a partial message from a previous write
	// in the buffer, skip the buffer and log directly.
	if w.buff.Len() == 0 {
		w.log(line, lvl, bodyidx)
		return
	}

	w.buff.Write(line)
	// recheck level
	if lvl == w.Level {
		lvl, _, bodyidx = w.Check(w.buff.Bytes())
	}

	// Log empty messages in the middle of the stream so that we don't lose
	// information when the user writes "foo\n\nbar".
	w.flush(true, lvl, bodyidx)

	return remaining
}

func (w *Writer) Close() error {
	return w.Sync()
}

// Sync flushes buffered data to the logger as a new log entry even if it
// doesn't contain a newline. This stage use default level.
func (w *Writer) Sync() error {
	w.flush(false, w.Level, 0)
	return nil
}

func isSpace(r rune) bool {
	return r == ' '
}

func (w *Writer) flush(allowEmpty bool, lvl zapcore.Level, bodyidx int) {
	if allowEmpty || w.buff.Len() > 0 {
		w.log(w.buff.Bytes(), lvl, bodyidx)
	}
	w.buff.Reset()
}

func (w *Writer) log(b []byte, lvl zapcore.Level, bodyidx int) {
	var msg string
	if bodyidx > 0 {
		msg = string(bytes.TrimLeftFunc(b[bodyidx:], isSpace))
	} else {
		msg = string(b)
	}

	if ce := w.Log.Check(lvl, msg); ce != nil {
		ce.Write()
	}
}
