package logx_test

import (
	"github.com/tsingsun/woocoo/core/conf"
	"github.com/tsingsun/woocoo/core/logx"
	"github.com/tsingsun/woocoo/testdata"
	"gopkg.in/natefinch/lumberjack.v2"
	"testing"
)

var config, _ = conf.NewWithOption(conf.LocalPath(testdata.Path("app.yaml")))

func init() {
	config.Load().AsGlobal()
}

func TestNewWithOption(t *testing.T) {
	type args struct {
		opt []logx.Option
	}
	tests := []struct {
		name string
		args args
	}{
		{"std",
			args{
				[]logx.Option{logx.Std(-1)},
			},
		},
		{"file",
			args{
				[]logx.Option{logx.LogRotate(&lumberjack.Logger{Filename: testdata.Path("testdata.log")}, -1)},
			},
		},
		{"multi",
			args{
				[]logx.Option{logx.LogRotate(&lumberjack.Logger{Filename: testdata.Path("testdata.log")}, -1), logx.Std(-1)},
			},
		},
		{"config",
			args{
				[]logx.Option{logx.Config(config), logx.BaseDir(testdata.Tmp(""))},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logx.NewWithOption(tt.args.opt...)
			logx.Infof("got info: %v", got)
		})
	}
}
