package conf

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/test/testdata"
)

func TestNew(t *testing.T) {
	type args struct {
		opt []Option
	}
	tests := []struct {
		name  string
		args  args
		check func(cnf *Configuration)
		panic bool
	}{
		{
			name: "default",
			args: args{opt: nil},
			check: func(cnf *Configuration) {
				assert.Equal(t, cnf.Development, false)
			},
		},
		{
			name: "local",
			args: args{opt: []Option{WithLocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			check: func(cnf *Configuration) {
				assert.Equal(t, cnf.opts.localPath, testdata.Path(testdata.DefaultConfigFile))
			},
		},
		{
			name: "local-ref",
			args: args{opt: []Option{WithLocalPath(filepath.Join("../../test/testdata", testdata.DefaultConfigFile))}},
			check: func(cnf *Configuration) {
				assert.Equal(t, cnf.opts.localPath, testdata.Path(testdata.DefaultConfigFile))
			},
		},
		{
			name:  "local no file",
			args:  args{opt: []Option{WithLocalPath(testdata.Path("xxxx"))}},
			check: nil,
			panic: true,
		},
		{
			name: "basedir",
			args: args{opt: []Option{WithBaseDir("."), WithLocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			check: func(cnf *Configuration) {
				_, currentFile, _, _ := runtime.Caller(0)
				assert.Equal(t, cnf.GetBaseDir(), path.Dir(currentFile))
				assert.Equal(t, cnf.opts.localPath, testdata.Path(testdata.DefaultConfigFile))
			},
		},
		{
			name: "attach",
			args: args{opt: []Option{
				WithBaseDir(testdata.BaseDir()),
				WithLocalPath(testdata.DefaultConfigFile),
				WithIncludeFiles(testdata.Path("etc/attach.yaml"))}},
			check: func(cnf *Configuration) {
				assert.Equal(t, cnf.opts.localPath, testdata.Path(testdata.DefaultConfigFile))
				assert.Equal(t, cnf.opts.includeFiles, []string{testdata.Path("etc/attach.yaml")})
			},
		},
		{
			name: "attach-nofile",
			args: args{opt: []Option{
				WithBaseDir(testdata.BaseDir()),
				WithLocalPath(testdata.DefaultConfigFile),
				WithIncludeFiles(testdata.Path("xxxx.yaml"))}},
			check: nil,
			panic: true,
		},
		{
			name: "global",
			args: args{opt: []Option{WithGlobal(true),
				WithLocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			check: func(cnf *Configuration) {
				cnf.parser = &Parser{}
				assert.Equal(t, Global().Configuration, cnf)
				assert.Equal(t, cnf.opts.global, true)
				assert.Equal(t, cnf.opts.localPath, testdata.Path(testdata.DefaultConfigFile))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				assert.Panics(t, func() {
					New(tt.args.opt...)
				})
				return
			}
			got := New(tt.args.opt...)
			tt.check(got)
		})
	}
}

func TestNewFromX(t *testing.T) {
	tests := []struct {
		name    string
		newFunc func() (*Configuration, error)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "new from bytes",
			newFunc: func() (*Configuration, error) {
				return NewFromBytes([]byte(`
namespace: tsingsun
appName: woocoo
version: 1.0.0
development: true
`)), nil
			},
			wantErr: assert.NoError,
		},
		{
			name: "new from bytes",
			newFunc: func() (*Configuration, error) {
				return NewFromStringMap(map[string]any{
					"namespace":   "tsingsun",
					"appName":     "woocoo",
					"version":     "1.0.0",
					"development": true,
				}), nil
			},
			wantErr: assert.NoError,
		},
		{
			name: "new from wrong file",
			newFunc: func() (*Configuration, error) {
				p, err := NewParserFromFile("err")
				return NewFromParse(p), err
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.Error(t, err)
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.newFunc()
			if !tt.wantErr(t, err) {
				return
			}
			got.Load()
			assert.Equal(t, "tsingsun", got.Namespace())
			assert.Equal(t, "woocoo", got.AppName())
			assert.Equal(t, "1.0.0", got.Version())
			assert.Equal(t, true, got.Development)
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

func TestConfiguration_Exists(t *testing.T) {
	cnf := New()
	assert.False(t, cnf.Exists())

	cnf = New(WithBaseDir(testdata.BaseDir()))
	assert.True(t, cnf.Exists())
}

func TestConfiguration_Load(t *testing.T) {
	type fields struct {
		cnf *Configuration
	}
	tests := []struct {
		name    string
		fields  fields
		require func(cnf *Configuration)
	}{
		{
			name: "merge",
			fields: fields{
				cnf: New(WithLocalPath(testdata.TestConfigFile()),
					WithBaseDir(testdata.BaseDir())),
			},
			require: func(cnf *Configuration) {
				cnf.Load()
				assert.Equal(t, "woocoo1", cnf.Get("appName"))
			},
		},
		{
			name: "env",
			fields: fields{
				cnf: New(WithLocalPath(testdata.Path("etc/env.yaml"))),
			},
			require: func(cnf *Configuration) {
				cnf.Load()
				ev, err := strconv.Atoi(os.Getenv("INT"))
				assert.NoError(t, err)
				assert.EqualValues(t, ev, cnf.Get("env.int"))
				assert.EqualValues(t, ev, cnf.Get("env.intSpace"))
				assert.EqualValues(t, ev, cnf.Get("env.intUnder"))
			},
		},
		{
			name: "attach error",
			fields: fields{
				cnf: NewFromStringMap(map[string]any{
					"includeFiles": []string{"err"},
				}),
			},
			require: func(cnf *Configuration) {
				assert.Panics(t, func() {
					cnf.Load()
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.require(tt.fields.cnf)
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
		dst any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "slice",
			fields: fields{parser: NewParserFromStringMap(map[string]any{"slice": []string{"a", "b", "c"}})},
			args:   args{key: "slice", dst: []string{}},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
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
		})
	}
}

func TestConfiguration_Merge(t *testing.T) {
	type fields struct {
		cnf *Configuration
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "merge",
			fields: fields{
				cnf: NewFromStringMap(map[string]any{
					"appName": "woocoo",
					"slice":   []string{"a", "b", "c"},
					"map":     map[string]any{"a": "a", "b": "b"},
				}),
			},
			args: args{
				b: []byte(`
appName: woocoo_merge
slice: [a, b]
map:
  a: a1
`),
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			fields: fields{
				cnf: NewFromStringMap(map[string]any{}),
			},
			args: args{
				b: []byte(`\t`),
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				assert.Error(t, err)
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.cnf
			err := c.Merge(tt.args.b)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, c.AppName(), "woocoo_merge")
			assert.Equal(t, c.StringSlice("slice"), []string{"a", "b"})
			assert.Equal(t, c.StringMap("map"), map[string]string{"a": "a1", "b": "b"})
			tt.wantErr(t, c.Merge(tt.args.b), fmt.Sprintf("Merge(%v)", tt.args.b))
		})
	}
}

func TestConfiguration_Static(t *testing.T) {
	t.Run("values", func(t *testing.T) {
		tm, err := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
		require.NoError(t, err)
		setting := map[string]any{
			"appName":     "woocoo",
			"StringSlice": []string{"a", "b", "c"},
			"Int":         1,
			"Bool":        true,
			"Float64":     1.0,
			"IntSlice":    []int{1, 2, 3},
			"StringMap": map[string]string{
				"a": "1",
			},
			"Time":      tm,
			"TimeStamp": tm.Unix(),
			"Duration":  "1s",
		}
		cnf := NewFromStringMap(setting)
		assert.Equal(t, cnf.AppName(), "woocoo")

		assert.True(t, IsSet("appName"))
		assert.False(t, IsSet("appname"))
		assert.Equal(t, String("appName"), "woocoo")
		assert.Equal(t, IntSlice("IntSlice"), []int{1, 2, 3})
		assert.Equal(t, Bool("Bool"), true)
		assert.Equal(t, Float64("Float64"), 1.0)
		assert.Equal(t, Int("Int"), 1)
		assert.Equal(t, Duration("Duration"), time.Second)
		assert.Equal(t, StringSlice("StringSlice"), []string{"a", "b", "c"})
		assert.Equal(t, StringMap("StringMap"), setting["StringMap"])
		assert.Equal(t, Time("Time", "2006-01-02 15:04:05 -0700 MST"), tm)
		assert.Equal(t, Time("TimeStamp", "").Unix(), tm.Unix())
		assert.Equal(t, AllSettings(), setting)

		assert.Equal(t, Get("appName"), "woocoo")
		assert.Nil(t, Get("appname"))
	})
	t.Run("path", func(t *testing.T) {
		_, currentFile, _, _ := runtime.Caller(0)
		dir := path.Dir(currentFile)
		cnf := New()
		cnf.SetBaseDir(".")
		assert.Equal(t, cnf.GetBaseDir(), ".")
		assert.Equal(t, Abs("path/file"), "path/file")
		assert.Equal(t, Abs("/path/file"), "/path/file")
		cnf.SetBaseDir(dir)
		assert.Equal(t, Abs("path/file"), path.Join(dir, "path/file"))
		assert.Equal(t, Abs("/path/file"), "/path/file")
	})
}

func TestConfiguration_Sub(t *testing.T) {
	type fields struct {
		cnf *Configuration
	}
	type args struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		check  func(got *Configuration)
		panic  bool
	}{
		{
			name: "sub",
			fields: fields{
				cnf: NewFromStringMap(map[string]any{
					"appName": "woocoo",
					"map": map[string]any{
						"a": map[string]any{"sub": "sub"},
						"b": 1,
					},
				}),
			},
			args: args{
				path: "map",
			},
			check: func(got *Configuration) {
				assert.Equal(t, got.AllSettings(), map[string]any{
					"a": map[string]any{"sub": "sub"},
					"b": 1,
				})
				gotsub := got.Sub("a")
				assert.Equal(t, got.Root(), gotsub.Root())
			},
		},
		{
			name: "empty path",
			fields: fields{
				cnf: NewFromStringMap(map[string]any{
					"appName": "woocoo",
				}),
			},
			args: args{
				path: "",
			},
			check: func(got *Configuration) {

			},
		},
		{
			name: "panic",
			fields: fields{
				cnf: NewFromStringMap(map[string]any{
					"appName": "woocoo",
				}),
			},
			args: args{
				path: "panic",
			},
			panic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.cnf
			if tt.panic {
				assert.Panics(t, func() {
					c.Sub(tt.args.path)
				})
				return
			}
			cnf := c.Sub(tt.args.path)
			if tt.args.path == "" {
				assert.Equal(t, c, cnf)
				return
			}
			assert.Equal(t, cnf.Root(), c)
			assert.Equal(t, cnf.opts, c.opts)
			assert.Equal(t, cnf.Development, c.Development)
			tt.check(cnf)
		})
	}
}

func TestConfiguration_Each(t *testing.T) {
	type fields struct {
		cnf *Configuration
	}
	var namelist []string
	type args struct {
		path string
		cb   func(name string, sub *Configuration)
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		check  func(keylist []string)
	}{
		{
			name: "each",
			fields: fields{
				cnf: NewFromBytes([]byte(`
path:
  groups:
  - group:
      key: value
`)),
			},
			args: args{
				path: "path.groups",
				cb: func(name string, sub *Configuration) {
					assert.Equal(t, "group", name)
					assert.Equal(t, "value", sub.String(Join("", "key")))
				},
			},
		},
		{
			name: "can't cut path",
			fields: fields{
				cnf: NewFromBytes([]byte(`
path:
  groups:
  - group:
    - slice:
        key: value
`)),
			},
			args: args{
				path: "path.groups",
				cb: func(name string, sub *Configuration) {
					assert.Equal(t, "group", name)
					assert.Len(t, sub.ParserOperator().Slices(name), 1, "should have one slice")
				},
			},
		},
		{
			name: "each",
			fields: fields{
				cnf: NewFromBytes([]byte(`
path:
  groups:
  - group:
      key: value
      key2: value2
`)),
			},
			args: args{
				path: "path.groups",
				cb: func(name string, sub *Configuration) {
					assert.Equal(t, "group", name)
					assert.Equal(t, "value", sub.String(Join("", "key")))
					assert.Equal(t, "value2", sub.String(Join("", "key2")))
				},
			},
		},
		{
			name: "keep order",
			fields: fields{
				cnf: NewFromBytes([]byte(`
path:
  groups:
  - zgroup:
      key: value
  - agroup:  
      key2: value2
  - dgroup:
      key3: value3
  - cgroup:
      key4: value4
`)),
			},
			args: args{
				path: "path.groups",
				cb: func(name string, sub *Configuration) {
					namelist = append(namelist, name)
				},
			},
			check: func(keylist []string) {
				assert.Equal(t, []string{"zgroup", "agroup", "dgroup", "cgroup"}, keylist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namelist = namelist[:0]
			c := tt.fields.cnf
			c.Each(tt.args.path, tt.args.cb)
			if tt.check != nil {
				tt.check(namelist)
			}
		})
	}
}

func TestConfiguration_Map(t *testing.T) {
	type fields struct {
		cnf *Configuration
	}
	type args struct {
		path string
		cb   func(name string, sub *Configuration)
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		panic  bool
	}{
		{
			name: "not map value",
			fields: fields{
				cnf: NewFromBytes([]byte(`
path:
  groups: 
    key: value
    key1: value1
`)),
			},
			args: args{
				path: "path.groups",
				cb: func(name string, sub *Configuration) {
					if !(name == "key" || name == "key1") {
						t.Error("name should be key or key1")
					}
				},
			},
			panic: true,
		},
		{
			name: "map",
			fields: fields{
				cnf: NewFromBytes([]byte(`
path:
  groups: 
    key: 
      key1: value1
`)),
			},
			args: args{
				path: "path.groups",
				cb: func(name string, sub *Configuration) {
					assert.Equal(t, "key", name)
					assert.Equal(t, "value1", sub.String("key1"))
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.cnf
			if tt.panic {
				assert.Panics(t, func() {
					c.Map(tt.args.path, tt.args.cb)
				})
				return
			}
			c.Map(tt.args.path, tt.args.cb)
		})
	}
}

func TestGlobal(t *testing.T) {
	tests := []struct {
		name    string
		want    func() *AppConfiguration
		wantErr bool
	}{
		{
			name: "global by conf new",
			want: func() *AppConfiguration {
				return &AppConfiguration{
					Configuration: New(WithGlobal(false)),
				}
			},
			wantErr: true,
		},
		{
			name: "global",
			want: func() *AppConfiguration {
				return &AppConfiguration{
					Configuration: New(),
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			global.Configuration = nil
			if tt.wantErr {
				assert.Panics(t, func() {
					Global()
				})
				return
			}
			assert.Equalf(t, tt.want(), Global(), "Global()")
		})
	}
}
