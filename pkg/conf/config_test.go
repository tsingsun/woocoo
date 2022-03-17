package conf

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	type args struct {
		opt []Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"default", args{opt: nil}, false},
		{"local",
			args{opt: []Option{LocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			false,
		},
		{"basedir",
			args{opt: []Option{BaseDir("."), LocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			false,
		},
		{"attach",
			args{opt: []Option{LocalPath(testdata.Path(testdata.DefaultConfigFile)), IncludeFiles(testdata.Path("config/attach.yaml"))}}, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			New()
		})
	}
}

func TestCopy(t *testing.T) {
	b := []byte(`
appname: woocoo
development: true
log:
  config:
    level: debug
duration: 1s
`)
	p, err := NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	cnf := New()
	cfg := cnf.CutFromParser(p)
	copyCfg := cfg.Copy()
	cfg.Parser().Set("appname", "woocoocopy")
	cfg.Parser().Set("log.config.level", "info")
	if copyCfg.Get("appname") == cfg.Get("appname") {
		t.Fatal()
	}
	if copyCfg.Duration("duration") != time.Second {
		t.Fatal("duration section copy error")
	}
}

func TestConfiguration_Load(t *testing.T) {
	type fields struct {
		cfg *Configuration
	}
	tests := []struct {
		name   string
		fields fields
		want   *Configuration
	}{
		{name: "merge", fields: fields{cfg: New(LocalPath(testdata.TestConfigFile()), BaseDir(testdata.BaseDir()))}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.fields.cfg.Load()
			if tt.name == "merge" {
				assert.Len(t, cfg.opts.includeFiles, 3) //slice merge no support
			}
		})
	}
}
