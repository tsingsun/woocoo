package log

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

// AsyncConfig configures asynchronous logging.
type AsyncConfig struct {
	// ChannelBuffer is the number of log entries to buffer in the channel.
	// If 0, defaults to 1024.
	ChannelBuffer int `json:"channelBuffer" yaml:"channelBuffer"`
}

// AsyncCore wraps a zapcore.Core to make Write calls asynchronous.
// Check (level filtering, sampling) remains synchronous to avoid
// queueing entries that would be discarded.
type AsyncCore struct {
	core     zapcore.Core
	entries  chan asyncEntry
	done     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

type asyncEntry struct {
	entry  zapcore.Entry
	fields []zapcore.Field
}

// NewAsyncCore creates a new AsyncCore wrapping the given core.
func NewAsyncCore(core zapcore.Core, bufSize int) *AsyncCore {
	if bufSize <= 0 {
		bufSize = 1024
	}
	ac := &AsyncCore{
		core:    core,
		entries: make(chan asyncEntry, bufSize),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go ac.worker()
	return ac
}

func (ac *AsyncCore) worker() {
	defer close(ac.stopped)
	for {
		select {
		case ae, ok := <-ac.entries:
			if !ok {
				return
			}
			ac.core.Write(ae.entry, ae.fields)
		case <-ac.done:
			// drain remaining entries
			for {
				select {
				case ae := <-ac.entries:
					ac.core.Write(ae.entry, ae.fields)
				default:
					return
				}
			}
		}
	}
}

// Enabled delegates to the underlying core (synchronous).
func (ac *AsyncCore) Enabled(lvl zapcore.Level) bool {
	return ac.core.Enabled(lvl)
}

// Check delegates to the underlying core for level/sampling decisions,
// but registers the async Write so entries are written asynchronously.
func (ac *AsyncCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if ac.core.Enabled(ent.Level) {
		return ce.AddCore(ent, ac)
	}
	return ce
}

// Write sends the entry to the worker goroutine asynchronously.
// If the channel is full, the entry is dropped to avoid blocking the caller.
func (ac *AsyncCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	select {
	case ac.entries <- asyncEntry{entry: ent, fields: fields}:
	default:
		// channel full, drop to avoid blocking
	}
	return nil
}

// With creates a child core with additional fields.
func (ac *AsyncCore) With(fields []zapcore.Field) zapcore.Core {
	return NewAsyncCore(ac.core.With(fields), cap(ac.entries))
}

// Sync stops the worker, drains pending entries, and syncs the underlying core.
func (ac *AsyncCore) Sync() error {
	ac.stopOnce.Do(func() {
		close(ac.done)
	})
	<-ac.stopped
	return ac.core.Sync()
}
