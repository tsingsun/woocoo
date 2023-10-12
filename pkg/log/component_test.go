package log

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/test/logtest"
	"go.uber.org/zap"
)

func TestComponent(t *testing.T) {
	type fields struct {
		logger ComponentLogger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "component-1", fields: fields{logger: Component("component-1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logdata := &logtest.Buffer{}
			zp := logtest.NewBuffLogger(logdata)
			l := &Logger{
				Logger: zp,
			}
			l.AsGlobal()
			got := tt.fields.logger
			assert.NotNil(t, got.Logger())
			got.SetLogger(l)
			got.Debug("debug", zap.String("key", "value"))
			got.Info("info", zap.String("key", "value"))
			got.Warn("warn", zap.String("key", "value"))
			got.Error("error", zap.String("key", "value"))
			got.DPanic("dpanic", zap.String("key", "value"))
			assert.Len(t, logdata.Lines(), 5)
		})
	}
}

func TestComponent_With(t *testing.T) {
	t.Run("WithOriginalLogger", func(t *testing.T) {
		logdata := &logtest.Buffer{}
		zp := logtest.NewBuffLogger(logdata)
		l := &Logger{
			Logger: zp,
		}
		l.AsGlobal()
		got := Global().Logger(WithOriginalLogger())
		assert.Equal(t, got, l)
		Component("component-log").SetLogger(got)
		Component("component-log").Debug("debug", zap.String("key", "value"))
		assert.Equal(t, 1, strings.Count(logdata.String(), `"component"`))
	})
	t.Run("NoWith-component-field-twice", func(t *testing.T) {
		logdata := &logtest.Buffer{}
		zp := logtest.NewBuffLogger(logdata)
		l := &Logger{
			Logger: zp,
		}
		l.AsGlobal()
		// will set "component" field twice
		Component("component-log").SetLogger(Global().Logger())
		Component("component-log").Debug("debug", zap.String("key", "value"))
		assert.Equal(t, 1, strings.Count(logdata.String(), `"component"`))
	})
	t.Run("WithContextLogger", func(t *testing.T) {
		logdata := &logtest.Buffer{}
		zp := logtest.NewBuffLogger(logdata)
		l := &Logger{
			Logger: zp,
		}
		l.AsGlobal()
		c := Component("component-log")
		c.SetLogger(l)
		got := Global().Logger(WithContextLogger())
		assert.NotEqual(t, got, l)
		got.Debug("debug", zap.String("key", "value"))
		assert.Equal(t, 0, strings.Count(logdata.String(), `"component"`))
		logdata.Reset()
		c.Debug("debug", zap.String("key", "value"))
		assert.Equal(t, 1, strings.Count(logdata.String(), `"component"`))
	})
}
