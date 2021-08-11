package conf_test

import (
	"bytes"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

// test instance package
// etcd required
//	docker run --restart=always --name etcd -p 2379:2379 -p 2380:2380 k8s.gcr.io/etcd:3.3.10 etcd
//	--listen-client-urls http://0.0.0.0:2379 --advertise-client-urls http://0.0.0.0:2380
func TestNew(t *testing.T) {
	type args struct {
		opt []conf.Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"default", args{opt: nil}, false},
		{"local",
			args{opt: []conf.Option{conf.LocalPath(testdata.Path(testdata.DefaultConfigFile))}},
			false,
		},
		//{"etcd",
		//	args{opt: []conf.Option{conf.RemoteProvider("etcd", "http://localhost:2379", "woocoo/test", "")}}, false,
		//},
		{"attach",
			args{opt: []conf.Option{conf.LocalPath(testdata.Path(testdata.DefaultConfigFile)), conf.IncludeFiles(testdata.Path("config/attach.yaml"))}}, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf.New()
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
    disableCaller: true
    disableStacktrace: true
    encoding: json
    outputPaths:
      - stdout
      - "test.log"
    errorOutputPaths:
      - stderr
`)
	p, err := conf.NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	cnf := conf.New()
	cfg := cnf.CutFromParser(p)
	copy := cfg.Copy()
	cfg.Parser().Set("appname", "woocoocopy")
	cfg.Parser().Set("log.config.level", "info")
	if copy.Get("appname") == cfg.Get("appname") {
		t.Fatal()
	}
}

//func TestConfig_WatchConfig(t *testing.T) {
//	endpoint := "127.0.0.1:2379"
//	path := "/woocoo/test/app.yaml"
//	cl, _ := clientv3.New(clientv3.Configuration{
//		Endpoints: []string{endpoint},
//	})
//	_, err := cl.Put(context.Background(), path, "appname: woocoo")
//	if err != nil {
//		t.Fatal(err)
//	}
//	type fields struct {
//		opts []conf.Option
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   string
//	}{
//		{"local", fields{
//			opts: []conf.Option{conf.LocalPath(testdata.Path("app.yaml"))},
//		}, ""},
//		{"etcd", fields{
//			opts: []conf.Option{conf.RemoteProvider("etcd", endpoint, path, "")},
//		}, "newapp"},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c, err := conf.BuildWithOption(tt.fields.opts...)
//			if err != nil {
//				t.Error(err)
//			}
//			c.BuildZap()
//			if err := c.WatchConfig(); err != nil {
//				t.Fatal(err)
//			}
//			time.Sleep(time.Second)
//			if tt.name == "etcd" {
//				_, err := cl.Put(context.Background(), path, "appname: "+tt.want)
//				if err != nil {
//					t.Fatal(err)
//				}
//				time.Sleep(3 * time.Second)
//				got := c.Parser().GetString("appname")
//				if got != tt.want {
//					t.Errorf("NewConfig() = %v, want %v", got, tt.want)
//				}
//			}
//		})
//	}
//}
