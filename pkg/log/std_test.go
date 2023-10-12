package log

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/test/logtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"testing"
)

func TestWriter(t *testing.T) {
	logdata := &logtest.Buffer{}
	zp := logtest.NewBuffLogger(logdata).WithOptions(zap.IncreaseLevel(zapcore.InfoLevel))
	w := Writer{
		Log:   zp,
		Level: zap.InfoLevel,
	}
	t.Run("normal", func(t *testing.T) {
		logdata.Reset()
		w.Write([]byte("test\n"))
		assert.Len(t, logdata.Lines(), 1)
		assert.Contains(t, logdata.String(), "test")
		assert.Contains(t, logdata.String(), "info")
	})
	t.Run("write part", func(t *testing.T) {
		logdata.Reset()
		w.Write([]byte("[DEBUG]test"))
		assert.Len(t, logdata.Lines(), 0)
		w.Write([]byte("\n"))
		assert.Len(t, logdata.Lines(), 1, "\\n should be a line")
		logdata.Reset()
		w.Write([]byte("[WARN]test"))
		w.Write([]byte("second part\n"))
		assert.Len(t, logdata.Lines(), 1)
		assert.Contains(t, logdata.String(), "second part")
		assert.Contains(t, logdata.String(), "warn")
	})
	t.Run("filter", func(t *testing.T) {
		logdata.Reset()
		_, err := w.Write([]byte("[DEBUG]test\n"))
		assert.NoError(t, err)
		assert.Equal(t, 0, len(logdata.Lines()))
	})
	t.Run("close", func(t *testing.T) {
		logdata.Reset()
		w.Write([]byte("[WARN]test"))
		assert.NoError(t, w.Close())
		assert.Len(t, logdata.Lines(), 1, "use info level, so should be a line")
		assert.Contains(t, logdata.String(), "info")
	})
	t.Run("cut", func(t *testing.T) {
		logdata.Reset()
		w.Write([]byte("2006/01/02 15:04:05 [INFO] it's a test\n"))
		assert.Len(t, logdata.Lines(), 1)
		assert.NotContains(t, logdata.String(), "2006/01/02 15:04:05")
		assert.Contains(t, logdata.String(), "it's a test", "should left trim space")
	})
}

func TestPrint(t *testing.T) {
	logdata := &logtest.Buffer{}
	zp := logtest.NewBuffLogger(logdata).WithOptions(zap.IncreaseLevel(zapcore.InfoLevel))
	w := Writer{
		Log:   zp,
		Level: zap.InfoLevel,
	}
	log.SetOutput(&w)
	Printf("test")
	assert.Len(t, logdata.Lines(), 1)
	logdata.Reset()
	Println("test")
	assert.Len(t, logdata.Lines(), 1)
}
