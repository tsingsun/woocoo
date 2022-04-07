package log

import (
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	zapConfigPath = "core"
	// rotate:[//[userinfo@]host][/]path[?query][#fragment]
	rotateSchema = "Rotate"
)

var once sync.Once

// Config is logger schema
// ZapConfigs use as zap advance,such as zapcore.NewTee()
// Sole use as one zap logger core
type Config struct {
	// ZapConfigs is for initial zap multi core
	ZapConfigs []zap.Config `json:"core" yaml:"core"`
	// Rotate is for log rotate
	Rotate *rotate `json:"rotate" yaml:"rotate"`
	// Disable automatic timestamps in output if use textEncoder
	DisableTimestamp bool `json:"disableTimestamp" yaml:"disableTimestamp"`
	// DisableErrorVerbose stops annotating logs with the full verbose error
	// message.
	DisableErrorVerbose bool `json:"disableErrorVerbose" yaml:"disableErrorVerbose"`

	useRotate bool
	basedir   string
}

type rotate struct {
	lumberjack.Logger `json:",squash" yaml:",squash"`
}

// Sync implement zap.Sink interface
//
// need nothing to do, see https://github.com/natefinch/lumberjack/pull/47
func (r *rotate) Sync() error {
	return nil
}

// NewConfig return a Config instance
func NewConfig(cfg *conf.Configuration) (*Config, error) {
	kps, err := cfg.SubOperator(zapConfigPath)
	if err != nil {
		panic(err)
	}
	if len(kps) == 0 {
		return nil, fmt.Errorf("none logger config,plz set up section: core")
	}
	v := &Config{
		ZapConfigs: make([]zap.Config, len(kps)),
		basedir:    cfg.Root().GetBaseDir(),
	}
	for i := 0; i < len(v.ZapConfigs); i++ {
		v.ZapConfigs[i] = defaultZapConfig(cfg)
	}

	if err = cfg.Unmarshal(&v); err != nil {
		return nil, err
	}

	if v.Rotate != nil {
		v.useRotate = true
	}
	return v, nil
}

// DefaultTimeEncoder serializes time.Time to a human-readable formatted string
func DefaultTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	s := t.Format("2006/01/02 15:04:05.000 -07:00")
	if e, ok := enc.(*textEncoder); ok {
		for _, c := range []byte(s) {
			e.buf.AppendByte(c)
		}
		return
	}
	enc.AppendString(s)
}

func defaultZapConfig(cfg *conf.Configuration) zap.Config {
	dzapCfg := zap.NewProductionConfig()
	//change default encode time format
	dzapCfg.EncoderConfig.EncodeTime = DefaultTimeEncoder
	dzapCfg.Development = cfg.Root().Development
	return dzapCfg
}

func (c Config) fixZapConfig(zc *zap.Config) error {
	var otps []string
	for _, path := range zc.OutputPaths {
		u, err := convertPath(path, c.basedir, c.useRotate)
		if err != nil {
			return err
		}
		otps = append(otps, u)
	}
	zc.OutputPaths = otps
	return nil
}

// BuildZap build a zap.Logger by Config
func (c *Config) BuildZap(opts ...zap.Option) (zl *zap.Logger, err error) {
	once.Do(func() {
		// register encoder
		encoder := NewTextEncoder(c)
		// text encode
		err = zap.RegisterEncoder("text", func(zapcore.EncoderConfig) (zapcore.Encoder, error) {
			return encoder, nil
		})
		if err != nil {
			panic(err)
		}

		//RegisterSink
		if c.useRotate {
			err := zap.RegisterSink(rotateSchema, func(u *url.URL) (zap.Sink, error) {
				if u.User != nil {
					return nil, fmt.Errorf("user and password not allowed with file URLs: got %v", u)
				}
				if u.Fragment != "" {
					return nil, fmt.Errorf("fragments not allowed with file URLs: got %v", u)
				}
				if u.RawQuery != "" {
					return nil, fmt.Errorf("query parameters not allowed with file URLs: got %v", u)
				}
				// Error messages are better if we check hostname and port separately.
				if u.Port() != "" {
					return nil, fmt.Errorf("ports not allowed with file URLs: got %v", u)
				}
				if hn := u.Hostname(); hn != "" && hn != "localhost" {
					return nil, fmt.Errorf("file URLs must leave host empty or use localhost: got %v", u)
				}
				l := c.newRotateWriter()
				if runtime.GOOS == "windows" {
					l.Filename = strings.TrimPrefix(u.Path, "/")
				} else {
					l.Filename = u.Path
				}
				return l, nil
			})
			if err != nil {
				panic(err)
			}
		}
	})

	var cores []zapcore.Core
	for _, zc := range c.ZapConfigs {
		if err = c.fixZapConfig(&zc); err != nil {
			return
		}
		tmpzl, err := zc.Build(opts...)
		if err != nil {
			return nil, err
		}
		cores = append(cores, tmpzl.Core())
	}
	if len(cores) == 1 {
		zl = zap.New(cores[0])
	} else {
		zl = zap.New(zapcore.NewTee(cores...))
	}
	return
}

func (c Config) newRotateWriter() *rotate {
	return &rotate{
		Logger: lumberjack.Logger{
			MaxSize:    c.Rotate.MaxSize,
			MaxAge:     c.Rotate.MaxAge,
			MaxBackups: c.Rotate.MaxBackups,
			LocalTime:  c.Rotate.LocalTime,
			Compress:   c.Rotate.Compress,
		},
	}
}

func convertPath(path string, base string, useRotate bool) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("can't parse %q as a URL: %v", path, err)
	}
	if path == "stdout" || path == "stderr" || (u.Scheme != "" && u.Scheme != "file") {
		return path, nil
	}
	if u.Scheme != "file" {
		var absPath string
		if !filepath.IsAbs(u.Path) {
			absPath = filepath.Join(base, path)
			if runtime.GOOS == "windows" {
				absPath = "/" + absPath
			}
			u.Path = absPath
		}
	}
	if useRotate {
		u.Scheme = rotateSchema
	}
	return u.String(), nil
}
