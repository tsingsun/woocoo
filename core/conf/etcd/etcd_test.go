// cat app.ymal | ETCDCTL_API=3 etcdctl  put woocoo/test/app.yaml
// docker run --rm k8s.gcr.io/etcd:3.3.10 sh -c "ETCDCTL_API=3 etcdctl --endpoints=192.168.31.34:2379 get woocoo/test/app.yaml"
package etcd

import (
	"bytes"
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/spf13/viper"
	"io/ioutil"
	"testing"
	"time"
)

func TestUseSimple(t *testing.T) {
	type args struct {
		connstring string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"empty", args{connstring: ""}, false},
		{"query", args{connstring: "/?username=a&password=b"}, false},
		{"all", args{connstring: "http://user:pwd@127.0.0.1:2379/?dial-timeout=30"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UseSimple(tt.args.connstring); (err != nil) != tt.wantErr {
				t.Errorf("UseSimple() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testRemoteProvider struct {
	provider      string
	endpoint      string
	path          string
	secretKeyring string
}

func (rp testRemoteProvider) Provider() string {
	return rp.provider
}

func (rp testRemoteProvider) Endpoint() string {
	return rp.endpoint
}

func (rp testRemoteProvider) Path() string {
	return rp.path
}

func (rp testRemoteProvider) SecretKeyring() string {
	return rp.secretKeyring
}

func Test_etcdConfigProvider_Get(t *testing.T) {
	tests := []struct {
		name    string
		args    viper.RemoteProvider
		wantErr bool
	}{
		{"Get", testRemoteProvider{"etcd", "http://127.0.0.1:2379", "/woocoo/test/app.yaml", ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := viper.RemoteConfig
			got, err := e.Get(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotStr, err := ioutil.ReadAll(got)
			if gotStr == nil {
				t.Errorf("Get() error = %v, wantErr %v", "nil", "app.yaml")
			}
		})
	}
}

func Test_etcdConfigProvider_WatchChannel(t *testing.T) {
	type fields struct {
		client *clientv3.Client
	}
	type args struct {
		rp viper.RemoteProvider
	}
	endPoint := "http://127.0.0.1:2379"
	cl, _ := clientv3.New(clientv3.Config{
		Endpoints: []string{endPoint},
	})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{"watch",
			fields{client: cl},
			args{testRemoteProvider{"etcd", endPoint, "/woocoo/test/app.yaml", ""}},
			"appname: newapp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := etcdConfigProvider{
				client: tt.fields.client,
			}
			got, done := e.WatchChannel(tt.args.rp)
			time.Sleep(time.Second)
			_, err := tt.fields.client.Put(context.Background(), tt.args.rp.Path(), tt.want)
			if err != nil {
				t.Fatal(err)
			}
			got1 := <-got
			if got1.Error != nil {
				t.Fatal(err)
			}
			r, err := ioutil.ReadAll(bytes.NewReader(got1.Value))
			if string(r) != tt.want || err != nil {
				t.Errorf("Get() error = %v, wantErr %v", string(r), tt.want)
			}
			done <- true
		})
	}
}
