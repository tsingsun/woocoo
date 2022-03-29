package log

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
	"strings"
	"testing"
)

func TestNewConfigSingle(t *testing.T) {
	var cfgStr = `
development: true
log:
  sole:
    level: debug
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
		useRotate: true,
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
	want.Sole = &zc
	assert.True(t, got.Sole.Development)
	assert.EqualValues(t, got.Sole.Level.Level(), want.Sole.Level.Level())
	assert.EqualValues(t, got.Sole.Encoding, want.Sole.Encoding)
	assert.EqualValues(t, got.Rotate, want.Rotate)
}

func TestNewConfigTee(t *testing.T) {
	var cfgStr = `
development: true
log:
  tee:
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
	assert.Len(t, got.Tee, 2)
	assert.Nil(t, got.Sole)
	for i, te := range got.Tee {
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
					zap.NewDevelopmentConfig(),
					{
						Level:         zap.NewAtomicLevelAt(zapcore.WarnLevel),
						Encoding:      "json",
						OutputPaths:   []string{t.Name() + "-warn.log"},
						EncoderConfig: zap.NewProductionEncoderConfig(),
					},
					{
						Level:         zap.NewAtomicLevelAt(zapcore.ErrorLevel),
						Encoding:      "json",
						OutputPaths:   []string{t.Name() + "-error.log"},
						EncoderConfig: zap.NewProductionEncoderConfig(),
					},
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
				Tee:       tt.fields.Tee,
				Sole:      tt.fields.Single,
				Rotate:    tt.fields.Rotate,
				useRotate: tt.fields.useRotate,
				basedir:   tt.fields.basedir,
			}
			gotZl, err := c.BuildZap(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildZap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(gotZl, tt.wantZl) {
			//	t.Errorf("BuildZap() gotZl = %v, want %v", gotZl, tt.wantZl)
			//}
			field := zap.String("logger", "test")
			gotZl.Debug(tt.name, field)
			gotZl.Info(tt.name, field)
			gotZl.Warn(tt.name, field)
			gotZl.Error(tt.name, field, zap.Error(fmt.Errorf("error")))
			gotZl.Sync()
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
						}
						//assert.NoError(t, os.Remove(lf))
					}
				}
			}

		})
	}
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
