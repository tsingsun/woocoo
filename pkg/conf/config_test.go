package conf

import (
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
			args{opt: []Option{WithLocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			false,
		},
		{"basedir",
			args{opt: []Option{WithBaseDir("."), WithLocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			false,
		},
		{"attach",
			args{opt: []Option{WithLocalPath(testdata.Path(testdata.DefaultConfigFile)), WithIncludeFiles(testdata.Path("etc/attach.yaml"))}}, false,
		},
		{
			"global",
			args{opt: []Option{WithGlobal(false), WithLocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			New(tt.args.opt...)
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
	cfg := NewFromBytes(b)
	copyCfg := cfg.Copy()
	cfg.Parser().Set("appname", "woocoocopy")
	cfg.Parser().Set("log.config.level", "info")
	assert.NotEqual(t, copyCfg.Get("appname"), cfg.Get("appname"))
	assert.Equal(t, copyCfg.Duration("duration"), time.Second)
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
		{name: "merge", fields: fields{cfg: New(WithLocalPath(testdata.TestConfigFile()), WithBaseDir(testdata.BaseDir()))}},
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

func TestConfiguration_Unmarshal(t *testing.T) {
	type fields struct {
		opts        options
		parser      *Parser
		Development bool
		root        *Configuration
	}
	type args struct {
		key string
		dst interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "slice",
			fields: fields{parser: NewParserFromStringMap(map[string]interface{}{"slice": []string{"a", "b", "c"}})},
			args:   args{key: "slice", dst: []string{}},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				assert.Len(t, i[0], 3)
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Configuration{
				opts:        tt.fields.opts,
				parser:      tt.fields.parser,
				Development: tt.fields.Development,
				root:        tt.fields.root,
			}
			tt.wantErr(t, c.parser.Unmarshal(tt.args.key, &tt.args.dst), tt.args.dst)
			//tt.wantErr(t, c.Sub(tt.args.key).Unmarshal(&tt.args.dst), tt.args.dst)
		})
	}
}
