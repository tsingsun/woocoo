package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func TestNewConfig(t *testing.T) {
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name    string
		args    args
		check   func(*Config)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "miss cores",
			args: args{
				cfg: conf.NewFromBytes([]byte("log:\n")),
			},
			check: func(cfg *Config) {
				assert.Nil(t, cfg)
			},
			wantErr: assert.Error,
		},
		{
			name: "default rotate",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
cores:
  - level: debug 
rotate:
  localtime: true
`)),
			},
			check: func(cfg *Config) {
				assert.True(t, cfg.useRotate, true)
			},
			wantErr: assert.NoError,
		},
		{
			name: "rotate default",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
cores:
  - level: debug
rotate:
`)),
			},
			check: func(cfg *Config) {
				assert.True(t, cfg.useRotate, true, "rotate lazy init, no need to assert values")
			},
			wantErr: assert.NoError,
		},
		{
			name: "rotate with nil sampling",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
cores:
  - level: debug
disableSampling: true
`)),
			},
			check: func(cfg *Config) {
				assert.Nil(t, cfg.ZapConfigs[0].Sampling)
			},
			wantErr: assert.NoError,
		},
		{
			name: "rotate with sampling",
			args: args{
				cfg: conf.NewFromBytes([]byte(`
cores:
  - level: debug
`)),
			},
			check: func(cfg *Config) {
				assert.Equal(t, 100, cfg.ZapConfigs[0].Sampling.Initial)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.args.cfg)
			if !tt.wantErr(t, err, fmt.Sprintf("NewConfig(%v)", tt.args.cfg)) {
				return
			}
			tt.check(got)
		})
	}
}

func TestNewConfigSolo(t *testing.T) {
	var cfgStr = `
development: true
log:
  withTraceID: true
  traceIDKey: trace_id
  cores:
    - level: debug
      disableCaller: true
      disableStacktrace: true
      encoding: json
      encoderConfig:
        timeEncoder: iso8601
      outputPaths:
        - stdout
      errorOutputPaths:
        - stderr
  rotate:
    maxSize: 1
    maxage: 1
    maxbackups: 1
    localtime: true
    compress: false
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).Load()
	got, err := NewConfig(cfg.Sub("log"))
	if err != nil {
		t.Error(err)
	}
	want := &Config{
		useRotate:  true,
		ZapConfigs: make([]zap.Config, 1),
		Rotate: &rotate{
			lumberjack.Logger{
				MaxSize:    1,
				MaxAge:     1,
				MaxBackups: 1,
				LocalTime:  true,
				Compress:   false,
			},
		},
	}
	zc := zap.NewProductionConfig()
	zc.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	zc.DisableCaller = true
	zc.DisableStacktrace = true
	zc.Encoding = "json"
	if err := zc.EncoderConfig.EncodeTime.UnmarshalText([]byte("iso8601")); err != nil {
		t.Error(err)
	}
	zc.OutputPaths = []string{"stdout"}
	want.ZapConfigs[0] = zc
	assert.True(t, got.WithTraceID)
	assert.Equal(t, "trace_id", got.TraceIDKey)
	assert.True(t, got.ZapConfigs[0].Development)
	assert.EqualValues(t, got.ZapConfigs[0].Level.Level(), want.ZapConfigs[0].Level.Level())
	assert.EqualValues(t, got.ZapConfigs[0].Encoding, want.ZapConfigs[0].Encoding)
	assert.EqualValues(t, got.Rotate, want.Rotate)
}

func TestNewConfigTee(t *testing.T) {
	var cfgStr = `
development: true
log:
  cores: 
    - level: debug 
      disableCaller: true
      disableStacktrace: true
      encoding: json
      encoderConfig:
        timeEncoder: iso8601
      outputPaths:
        - stdout
        - "test.log"
      errorOutputPaths:
        - stderr
    - level: warn 
      disableCaller: true
      outputPaths: 
        - "test.log"
      errorOutputPaths:
        - stderr      
  rotate:
    maxSize: 1
    maxage: 1
    maxbackups: 1
    localtime: true
    compress: false
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).Load()
	got, err := NewConfig(cfg.Sub("log"))
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, got.ZapConfigs, 2)
	for i, te := range got.ZapConfigs {
		if i == 0 {
			assert.Equal(t, zap.NewAtomicLevelAt(zapcore.DebugLevel).Level(), te.Level.Level())
		} else if i == 1 {
			assert.Equal(t, zap.NewAtomicLevelAt(zapcore.WarnLevel).Level(), te.Level.Level())
		}
	}
}

func TestConfig_BuildZap(t *testing.T) {
	type fields struct {
		Tee       []zap.Config
		Single    *zap.Config
		Rotate    *rotate
		useRotate bool
		basedir   string
	}
	type args struct {
		opts []zap.Option
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantZl  *zap.Logger
		wantErr bool
	}{
		{
			name: "tee",
			fields: fields{
				Tee: []zap.Config{
					{
						Level:             zap.NewAtomicLevelAt(zapcore.WarnLevel),
						Encoding:          "json",
						DisableStacktrace: false,
						OutputPaths:       []string{t.Name() + "-warn.log"},
						EncoderConfig:     zap.NewProductionEncoderConfig(),
					},
					{
						Level:         zap.NewAtomicLevelAt(zapcore.ErrorLevel),
						Encoding:      "json",
						OutputPaths:   []string{t.Name() + "-error.log"},
						EncoderConfig: zap.NewProductionEncoderConfig(),
					},
					zap.NewDevelopmentConfig(),
				},
				Single: nil,
				Rotate: &rotate{
					lumberjack.Logger{
						MaxSize:    1,
						MaxAge:     1,
						MaxBackups: 1,
						LocalTime:  true,
						Compress:   false,
					},
				},
				useRotate: true,
				basedir:   testdata.Tmp(""),
			},
			args:    args{opts: nil},
			wantZl:  nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ZapConfigs: tt.fields.Tee,
				Rotate:     tt.fields.Rotate,
				useRotate:  tt.fields.useRotate,
				basedir:    tt.fields.basedir,
			}
			gotZl, err := c.BuildZap(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildZap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			field := zap.String("logger", "test")
			gotZl.Debug(tt.name, field)
			gotZl.Info(tt.name, field)
			gotZl.Warn(tt.name, field)
			gotZl.Error(tt.name, field, zap.Error(fmt.Errorf("error")))
			_ = gotZl.Sync()
			for _, tee := range tt.fields.Tee {
				for _, outputPath := range tee.OutputPaths {
					if outputPath == "stdout" || outputPath == "stderr" {
						continue
					}
					if lf := testdata.Tmp(outputPath); path.IsAbs(lf) {
						bs, err := os.Open(lf)
						assert.NoError(t, err)
						lc, err := lineCounter(bs)
						assert.NoError(t, err)
						if strings.Index(outputPath, "warn") > 0 {
							assert.Equal(t, 2, lc)
						} else if strings.Index(outputPath, "error") > 0 {
							assert.Equal(t, 1, lc)
							if !tt.fields.Tee[0].DisableStacktrace {
								bs, _ := os.Open(lf)
								c, err := io.ReadAll(bs)
								require.NoError(t, err)
								assert.Contains(t, string(c), "stacktrace")
							}
						}
						assert.NoError(t, os.Remove(lf))
					}
				}
			}
		})
	}
}

func TestTextEncode(t *testing.T) {
	var cfgStr = `
development: true
log:
  disableTimestamp: true
  disableErrorVerbose: true
  cores:
    - level: debug
      disableCaller: true
      disableStacktrace: true
      encoding: text
`
	cfg := conf.NewFromBytes([]byte(cfgStr)).Load()
	got, err := NewConfig(cfg.Sub("log"))
	if err != nil {
		t.Error(err)
	}
	assert.True(t, got.DisableTimestamp)
	assert.True(t, got.DisableErrorVerbose)
	assert.Equal(t, "text", got.ZapConfigs[0].Encoding)
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
