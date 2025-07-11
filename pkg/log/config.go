package log

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	zapConfigPath = "cores"
	// rotate:[//[userinfo@]host][/]path[?query][#fragment]
	rotateSchema = "Rotate"

	StacktraceKey = "stacktrace"
	CallerSkip    = 1

	TraceIDKey = "trace_id"

	ComponentKey      = "component"
	WebComponentName  = "web"
	GrpcComponentName = "grpc"
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
	// DisableSampling disables sampling for all the loggers
	DisableSampling bool `json:"disableSampling" yaml:"disableSampling"`
	// Disable automatic timestamps in output if use TextEncoder
	DisableTimestamp bool `json:"disableTimestamp" yaml:"disableTimestamp"`
	// DisableErrorVerbose stops annotating logs with the full verbose error
	// message.
	DisableErrorVerbose bool `json:"disableErrorVerbose" yaml:"disableErrorVerbose"`
	// WithTraceID configures the logger to add `trace_id` field to structured log messages.
	WithTraceID bool `json:"withTraceID" yaml:"withTraceID"`
	// TraceIDKey is the key used to store the trace ID. defaults to "trace_id".
	TraceIDKey string `json:"traceIDKey" yaml:"traceIDKey"`
	callerSkip int
	useRotate  bool
	basedir    string
}

type rotate struct {
	// mapstructor use ",squash" tag for embedded struct, but conf.decoderConfig use `squash=true` so need not set
	lumberjack.Logger `json:",inline" yaml:",inline"`
}

// Sync implement zap.Sink interface
//
// needs nothing to do, see https://github.com/natefinch/lumberjack/pull/47
func (r *rotate) Sync() error {
	return nil
}

// NewConfig return a Config instance
func NewConfig(cnf *conf.Configuration) (*Config, error) {
	coresl := len(cnf.ParserOperator().Slices(zapConfigPath))
	if coresl == 0 {
		return nil, fmt.Errorf("none logger config,plz set up section: cores")
	}
	v := &Config{
		ZapConfigs: make([]zap.Config, coresl),
		basedir:    cnf.Root().GetBaseDir(),
		callerSkip: CallerSkip,
		TraceIDKey: TraceIDKey,
	}
	if cs := "callerSkip"; cnf.IsSet(cs) {
		v.callerSkip = cnf.Int(cs)
	}
	for i := 0; i < len(v.ZapConfigs); i++ {
		v.ZapConfigs[i] = defaultZapConfig(cnf)
	}

	if err := cnf.Unmarshal(&v); err != nil {
		return nil, err
	}

	if v.Rotate != nil || cnf.IsSet("rotate") {
		v.useRotate = true
	}
	return v, nil
}

// DefaultTimeEncoder serializes time.Time to a human-readable formatted string
func DefaultTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	s := t.Format("2006/01/02 15:04:05.000 -07:00")
	if e, ok := enc.(*TextEncoder); ok {
		for _, c := range []byte(s) {
			e.buf.AppendByte(c)
		}
		return
	}
	enc.AppendString(s)
}

func defaultZapConfig(cnf *conf.Configuration) zap.Config {
	dzapCfg := zap.NewProductionConfig()
	// change default encode time format
	dzapCfg.EncoderConfig.EncodeTime = DefaultTimeEncoder
	dzapCfg.Development = cnf.Root().Development
	if cnf.Bool("disableSampling") {
		dzapCfg.Sampling = nil
	}
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

// BuildZap build a zap.Logger by Config.
// Multi Zap Config not means multi loggers. It collects all zap cores and build a zap.Logger.
func (c *Config) BuildZap(opts ...zap.Option) (zl *zap.Logger, err error) {
	once.Do(func() {
		// register encoder
		encoder := buildTextEncoder(c)
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
	zl = zap.New(zapcore.NewTee(cores...), opts...)
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

func convertPath(path string, base string, useRotate bool) (cp string, err error) {
	isFile := false
	defer func() {
		if isFile {
			err = os.MkdirAll(filepath.Dir(cp), 0755)
			if useRotate {
				if runtime.GOOS == "windows" {
					cp = rotateSchema + ":///" + cp
				} else {
					cp = rotateSchema + "://" + cp
				}
			}
		}
	}()
	if path == "stdout" || path == "stderr" {
		return path, nil
	}
	if filepath.IsAbs(path) {
		isFile = true
		return path, nil
	}
	uri, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("can't parse %q as a URL: %v", path, err)
	}
	if uri.Scheme == "" {
		// ref path
		cp = filepath.Join(base, path)
		isFile = true
		return
	}
	uri.Scheme = rotateSchema
	return uri.String(), nil
}
