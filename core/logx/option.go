package logx

import (
	"fmt"
	"github.com/tsingsun/woocoo/core/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

type Option func(*options)

type options struct {
	config  *conf.Config
	cores   []zapcore.Core
	basedir string
	useStd  bool
}

var defaultOptions = options{
	cores:   []zapcore.Core{},
	basedir: "logs",
}

func getLevel(cLevel int) zapcore.Level {
	level := zap.InfoLevel
	if cLevel >= -1 && cLevel <= 5 {
		level = levelConv(int8(cLevel))
	}
	return level
}

func levelConv(level int8) zapcore.Level {
	switch level {
	case -1:
		return zap.DebugLevel
	case 0:
		return zap.InfoLevel
	case 1:
		return zap.WarnLevel
	case 2:
		return zap.ErrorLevel
	case 3:
		return zap.DPanicLevel
	case 4:
		return zap.PanicLevel
	case 5:
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func Config(config *conf.Config) Option {
	return func(o *options) {
		o.config = config
	}
}

//load from application configuration
//base dir is log file directory
//base dir must add before LogRotate,otherwise LogRotate's filename may be causing an exception
func BaseDir(basedir string) Option {
	return func(o *options) {
		o.basedir = basedir
	}
}

// lumberjack.Logger is already safe for concurrent use, so we don't need to
// lock it.
// level: debug:-1;info:0;warning:1;error:2;DPanic:3;Panic:4;Fatal:5
func LogRotate(ll *lumberjack.Logger, level int) Option {
	return func(o *options) {
		if o.basedir != "" {
			if !filepath.IsAbs(ll.Filename) {
				ll.Filename = filepath.Join(o.basedir, ll.Filename)
			}
		}
		fp := ll.Filename
		if _, err := os.Stat(fp); err != nil {
			if os.MkdirAll(filepath.Dir(fp), os.FileMode(0755)) != nil {
				panic(fmt.Sprintf("invalid logger filename: %s", fp))
			}
			fi, err := os.Create(fp)
			if err != nil {
				panic(fmt.Sprintf("create Logger filename: %s failure", fp))
			}
			defer fi.Close()
		}

		var ec zapcore.EncoderConfig
		if o.config != nil && o.config.IsDebug() {
			ec = zap.NewDevelopmentEncoderConfig()
		} else {
			ec = zap.NewProductionEncoderConfig()
		}

		w := zapcore.AddSync(ll)
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(ec),
			w,
			getLevel(level),
		)
		o.cores = append(o.cores, core)
	}
}

func Std(level int) Option {
	return func(o *options) {
		var ec zapcore.EncoderConfig
		if o.config != nil && o.config.IsDebug() {
			ec = zap.NewDevelopmentEncoderConfig()
		} else {
			ec = zap.NewProductionEncoderConfig()
		}
		consoleEncoder := zapcore.NewConsoleEncoder(ec)
		consoleDebugging := zapcore.Lock(os.Stdout)
		core := zapcore.NewCore(consoleEncoder, consoleDebugging, getLevel(level))
		o.cores = append(o.cores, core)
		o.useStd = true
	}
}
