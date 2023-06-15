package log

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tsingsun/woocoo/testco/logtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	errExample = errors.New("fail")

	_messages   = fakeMessages(1000)
	_tenInts    = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	_tenStrings = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	_tenTimes   = []time.Time{
		time.Unix(0, 0),
		time.Unix(1, 0),
		time.Unix(2, 0),
		time.Unix(3, 0),
		time.Unix(4, 0),
		time.Unix(5, 0),
		time.Unix(6, 0),
		time.Unix(7, 0),
		time.Unix(8, 0),
		time.Unix(9, 0),
	}
	_oneUser = &user{
		Name:      "Jane Doe",
		Email:     "jane@test.com",
		CreatedAt: time.Date(1980, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	_tenUsers = users{
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
		_oneUser,
	}
)

func fakeMessages(n int) []string {
	messages := make([]string, n)
	for i := range messages {
		messages[i] = fmt.Sprintf("Test logging, but use a somewhat realistic message length. (#%v)", i)
	}
	return messages
}

func getMessage(iter int) string {
	return _messages[iter%1000]
}

func newZapLogger(lvl zapcore.Level) *zap.Logger {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.NanosDurationEncoder
	ec.EncodeTime = zapcore.EpochNanosTimeEncoder
	enc := zapcore.NewJSONEncoder(ec)
	return zap.New(zapcore.NewCore(
		enc,
		&logtest.Discarder{},
		lvl,
	))
}

func newWcLogger(lvl zapcore.Level) *Logger {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.NanosDurationEncoder
	ec.EncodeTime = zapcore.EpochNanosTimeEncoder
	enc := zapcore.NewJSONEncoder(ec)
	return New(zap.New(zapcore.NewCore(
		enc,
		&logtest.Discarder{},
		lvl,
	))).AsGlobal()
}

func fakeFields() []zap.Field {
	return []zap.Field{
		zap.Int("int", _tenInts[0]),
		zap.Ints("ints", _tenInts),
		zap.String("string", _tenStrings[0]),
		zap.Strings("strings", _tenStrings),
		zap.Time("time", _tenTimes[0]),
		zap.Times("times", _tenTimes),
		zap.Object("user1", _oneUser),
		zap.Object("user2", _oneUser),
		zap.Array("users", _tenUsers),
		zap.Error(errExample),
	}
}

func BenchmarkZap(b *testing.B) {
	b.Logf("Logging with additional context at each log site.")
	b.Run("Zap", func(b *testing.B) {
		b.ReportAllocs()
		logger := newZapLogger(zap.DebugLevel)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info(getMessage(0), fakeFields()...)
			}
		})
	})
}

func BenchmarkWc(b *testing.B) {
	b.Logf("Logging with additional context at each log site.")
	b.Run("woocoo", func(b *testing.B) {
		b.ReportAllocs()
		newWcLogger(zap.DebugLevel)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				global.Info(getMessage(0), fakeFields()...)
			}
		})
	})
}

func BenchmarkWcWithContext(b *testing.B) {
	b.Logf("Logging with additional context at each log site.")
	b.Run("woocoo", func(b *testing.B) {
		b.ReportAllocs()
		newWcLogger(zap.DebugLevel)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				global.Ctx(context.Background()).Info(getMessage(0), fakeFields()...)
			}
		})
	})
}
