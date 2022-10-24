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
	zapConfigPath = "cores"
	// rotate:[//[userinfo@]host][/]path[?query][#fragment]
	rotateSchema = "Rotate"

	StacktraceKey = "stacktrace"
	CallerSkip    = 1

	TraceIDKey = "trace_id"

	ComponentKey     = "component"
	WebComponentName = "web"
)

var once sync.Once

// Config is logger schema
// ZapConfigs use as zap advance,such as zapcore.NewTee()
// Sole use as one zap logger core
type Config struct {
	// ZapConfigs is for initial zap multi core
	ZapConfigs []zap.Config `json:"cores" yaml:"cores"`
	// Rotate is for log rotate
	Rotate *rotate `json:"rotate" yaml:"rotate"`
	// Disable automatic timestamps in output if use textEncoder
	DisableTimestamp bool `json:"disableTimestamp" yaml:"disableTimestamp"`
	// DisableErrorVerbose stops annotating logs with the full verbose error
	// message.
	DisableErrorVerbose bool `json:"disableErrorVerbose" yaml:"disableErrorVerbose"`
	// WithTraceID configures the logger to add `trace_id` field to structured log messages.
	WithTraceID bool `json:"withTraceID" yaml:"withTraceID"`

	callerSkip int
	useRotate  bool
	basedir    string
}

type rotate struct {
	// mapstructor use json tag,so ignore `unknown JSON option "squash"` lint
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
	coresl := len(cfg.ParserOperator().Slices(zapConfigPath))
	if coresl == 0 {
		return nil, fmt.Errorf("none logger config,plz set up section: cores")
	}
	v := &Config{
		ZapConfigs: make([]zap.Config, coresl),
		basedir:    cfg.Root().GetBaseDir(),
		callerSkip: CallerSkip,
	}
	if cfg.IsSet("callerSkip") {
		v.callerSkip = cfg.Int("callerSkip")
	}
	for i := 0; i < len(v.ZapConfigs); i++ {
		v.ZapConfigs[i] = defaultZapConfig(cfg)
	}

	if err := cfg.Unmarshal(&v); err != nil {
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
	// change default encode time format
	dzapCfg.EncoderConfig.EncodeTime = DefaultTimeEncoder
	dzapCfg.Development = cfg.Root().Development
	return dzapCfg
}

func (c *Config) fixZapConfig(zc *zap.Config) error {
	otps := make([]string, len(zc.OutputPaths))
	for i, path := range zc.OutputPaths {
		u, err := convertPath(path, c.basedir, c.useRotate)
		if err != nil {
			return err
		}
		otps[i] = u
	}
	zc.OutputPaths = otps
	zc.EncoderConfig.StacktraceKey = StacktraceKey
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

		// RegisterSink
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

	var (
		cores []zapcore.Core
		copts []zap.Option
	)
	for i := range c.ZapConfigs {
		zc := c.ZapConfigs[i]
		err = c.fixZapConfig(&zc)
		if err != nil {
			return nil, err
		}
		tmpzl, err := zc.Build()
		if err != nil {
			return nil, err
		}
		cores = append(cores, tmpzl.Core())
		if i == 0 {
			copts = c.buildZapOptions(&zc)
		}
	}
	opts = append(opts, copts...)
	if len(cores) == 1 {
		zl = zap.New(cores[0], opts...)
	} else {
		zl = zap.New(zapcore.NewTee(cores...), opts...)
	}
	return
}

func (c *Config) buildZapOptions(cfg *zap.Config) (opts []zap.Option) {
	if !cfg.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}

	stackLevel := zap.ErrorLevel
	if cfg.Development {
		stackLevel = zap.WarnLevel
	}
	if !cfg.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}

	return opts
}

func (c *Config) newRotateWriter() *rotate {
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
