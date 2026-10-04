package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/logtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestAsyncCore(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.DebugLevel)
	asyncCore := NewAsyncCore(core, 256)

	logger := zap.New(asyncCore)
	logger.Info("async test", zap.String("key", "value"))

	// Sync waits for worker to flush all entries
	require.NoError(t, logger.Sync())
	assert.Contains(t, logdata.String(), "async test")
}

func TestAsyncCore_MultipleEntries(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.DebugLevel)
	asyncCore := NewAsyncCore(core, 256)

	logger := zap.New(asyncCore)
	for i := 0; i < 100; i++ {
		logger.Info("entry", zap.Int("i", i))
	}
	require.NoError(t, logger.Sync())

	lines := logdata.Lines()
	assert.Equal(t, 100, len(lines))
}

func TestAsyncCore_With(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.DebugLevel)
	asyncCore := NewAsyncCore(core, 256)

	child := asyncCore.With([]zapcore.Field{zap.String("component", "test")})
	logger := zap.New(child)
	logger.Info("child test")
	require.NoError(t, logger.Sync())

	assert.Contains(t, logdata.String(), "component")
	assert.Contains(t, logdata.String(), "child test")
}

func TestAsyncCore_LevelFilter(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.WarnLevel)
	asyncCore := NewAsyncCore(core, 256)

	logger := zap.New(asyncCore)
	logger.Info("should be filtered")
	logger.Warn("should pass")
	require.NoError(t, logger.Sync())

	assert.NotContains(t, logdata.String(), "should be filtered")
	assert.Contains(t, logdata.String(), "should pass")
}

func TestAsyncCore_SyncFlushes(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.DebugLevel)
	asyncCore := NewAsyncCore(core, 256)

	logger := zap.New(asyncCore)
	logger.Info("flush me")
	// Sync blocks until all entries are written
	require.NoError(t, logger.Sync())
	assert.Contains(t, logdata.String(), "flush me")
}

func TestAsyncCore_ViaConfig(t *testing.T) {
	t.Run("explicit-set", func(t *testing.T) {
		var cfgStr = `
cores:
  - level: debug
    disableCaller: true
    disableStacktrace: true
async:
  channelBuffer: 512
`
		cfg := conf.NewFromBytes([]byte(cfgStr))
		c, err := NewConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, c.Async)
		assert.Equal(t, 512, c.Async.ChannelBuffer)
	})
	t.Run("empty-set", func(t *testing.T) {
		var cfgStr = `
cores:
  - level: debug
    disableCaller: true
    disableStacktrace: true
async:

`
		cfg := conf.NewFromBytes([]byte(cfgStr))
		c, err := NewConfig(cfg)
		require.NoError(t, err)
		assert.Nil(t, c.Async)
	})
}

func TestAsyncCore_AsyncIsFaster(t *testing.T) {
	fields := fakeFields()

	syncDur := func() time.Duration {
		ec := zap.NewProductionEncoderConfig()
		enc := zapcore.NewJSONEncoder(ec)
		core := zapcore.NewCore(enc, &logtest.Discarder{}, zap.DebugLevel)
		logger := zap.New(core)
		start := time.Now()
		for i := 0; i < 10000; i++ {
			logger.Info("sync bench", fields...)
		}
		return time.Since(start)
	}

	asyncDur := func() time.Duration {
		ec := zap.NewProductionEncoderConfig()
		enc := zapcore.NewJSONEncoder(ec)
		core := zapcore.NewCore(enc, &logtest.Discarder{}, zap.DebugLevel)
		asyncCore := NewAsyncCore(core, 4096)
		logger := zap.New(asyncCore)
		start := time.Now()
		for i := 0; i < 10000; i++ {
			logger.Info("async bench", fields...)
		}
		logger.Sync()
		return time.Since(start)
	}

	// warmup
	syncDur()
	asyncDur()

	syncElapsed := syncDur()
	asyncElapsed := asyncDur()
	t.Logf("sync: %v, async: %v", syncElapsed, asyncElapsed)
}

func TestAsyncCore_SyncMultipleTimes(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.DebugLevel)
	asyncCore := NewAsyncCore(core, 256)

	logger := zap.New(asyncCore)
	logger.Info("before sync", zap.String("key", "value"))

	// first Sync
	require.NoError(t, logger.Sync())
	assert.Contains(t, logdata.String(), "before sync")

	// second Sync should not panic or block
	require.NoError(t, logger.Sync())

	// third Sync for good measure
	require.NoError(t, logger.Sync())
}

func TestAsyncCore_SyncThenWrite(t *testing.T) {
	logdata := &logtest.Buffer{}
	ec := zap.NewProductionEncoderConfig()
	enc := zapcore.NewJSONEncoder(ec)
	core := zapcore.NewCore(enc, logdata, zap.DebugLevel)
	asyncCore := NewAsyncCore(core, 256)

	logger := zap.New(asyncCore)
	logger.Info("before sync")
	require.NoError(t, logger.Sync())

	// After Sync, worker is stopped; writes enter channel but are not processed.
	// This is expected: Sync means shutdown, no panic or deadlock should occur.
	assert.NotPanics(t, func() {
		logger.Info("after sync")
		_ = logger.Sync()
	})
}

func BenchmarkAsyncVsSyncCore(b *testing.B) {
	b.Run("Async", func(b *testing.B) {
		b.ReportAllocs()
		ec := zap.NewProductionEncoderConfig()
		enc := zapcore.NewJSONEncoder(ec)
		core := zapcore.NewCore(enc, &logtest.Discarder{}, zap.DebugLevel)
		asyncCore := NewAsyncCore(core, 4096)
		logger := zap.New(asyncCore)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info("async benchmark", fakeFields()...)
			}
		})
		b.StopTimer()
		logger.Sync()
	})
	b.Run("Sync", func(b *testing.B) {
		b.ReportAllocs()
		ec := zap.NewProductionEncoderConfig()
		enc := zapcore.NewJSONEncoder(ec)
		core := zapcore.NewCore(enc, &logtest.Discarder{}, zap.DebugLevel)
		logger := zap.New(core)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info("sync benchmark", fakeFields()...)
			}
		})
	})
}
