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
)

const (
	teeConfigPath = "tee"
	// rotate:[//[userinfo@]host][/]path[?query][#fragment]
	rotateSchema = "Rotate"
)

var once sync.Once

// Config is logger schema
// Tee use as zap advance,such as zapcore.NewTee()
// Sole use as one zap logger core
type Config struct {
	Tee    []zap.Config `json:"tee" yaml:"tee"`
	Sole   *zap.Config  `json:"sole" yaml:"sole"`
	Rotate *rotate      `json:"rotate" yaml:"rotate"`

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
	kps, err := cfg.SubOperator(teeConfigPath)
	if err != nil {
		panic(err)
	}
	v := &Config{
		Tee:     make([]zap.Config, len(kps)),
		basedir: cfg.Root().GetBaseDir(),
	}
	dzapCfg := defaultZapConfig(cfg)

	if len(v.Tee) == 0 {
		v.Sole = &dzapCfg
	} else {
		for i := 0; i < len(v.Tee); i++ {
			v.Tee[i] = defaultZapConfig(cfg)
		}
	}
	if err = cfg.Unmarshal(&v); err != nil {
		return nil, err
	}

	if len(v.Tee) == 0 && v.Sole == nil {
		return nil, fmt.Errorf("none logger config,plz set up section: sole or tee")
	}

	if v.Rotate != nil {
		v.useRotate = true
	}
	if len(v.Tee) != 0 && v.Sole != nil {
		StdPrintln("single logger config is ineffective if using tee logger")
		v.Sole = nil
	}
	return v, nil
}

func defaultZapConfig(cfg *conf.Configuration) zap.Config {
	dzapCfg := zap.NewProductionConfig()
	//change default encode time format
	dzapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
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
	if c.useRotate {
		once.Do(func() {
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
		})
	}

	if c.Sole != nil {
		if err = c.fixZapConfig(c.Sole); err != nil {
			return
		}
		zl, err = c.Sole.Build(opts...)
		return
	}

	var cores []zapcore.Core
	for _, zc := range c.Tee {
		if err = c.fixZapConfig(&zc); err != nil {
			return
		}
		tmpzl, err := zc.Build(opts...)
		if err != nil {
			return nil, err
		}
		cores = append(cores, tmpzl.Core())
	}
	cr := zapcore.NewTee(cores...)
	zl = zap.New(cr)
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
