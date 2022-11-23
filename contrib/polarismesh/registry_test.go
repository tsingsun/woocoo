package polarismesh

import (
	"github.com/polarismesh/polaris-go/api"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
	"time"
)

func TestRegistry_Apply(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: :20000
    nameSpace: /woocoo/service
    version: "1.0"
    ipv6: true
  registry:
    scheme: polaris
    ttl: 600s
    polaris:
      configFile: etc/polaris/config.yaml
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
`)
	cfg := conf.NewFromBytes(b)

	type fields struct {
		opts            Options
		registerContext *RegisterContext
	}
	type args struct {
		cfg *conf.Configuration
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "ok", fields: fields{registerContext: &RegisterContext{}}, args: args{cfg: cfg.Sub("grpc.registry")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Registry{
				opts:            tt.fields.opts,
				registerContext: tt.fields.registerContext,
			}
			r.Apply(tt.args.cfg)
		})
	}
}

func TestRegistry_Register(t *testing.T) {
	type fields struct {
		opts            Options
		registerContext *RegisterContext
	}
	type args struct {
		serviceInfo *registry.ServiceInfo
		testType    int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "registerAndUnregister",
			fields: fields{opts: Options{TTL: time.Second * 600}, registerContext: func() *RegisterContext {
				var err error
				ctx := &RegisterContext{}
				ctx.providerAPI, err = api.NewProviderAPIByFile(testdata.Path("etc/polaris/polaris.yaml"))
				assert.NoError(t, err)
				return ctx
			}()},
			args: args{
				serviceInfo: &registry.ServiceInfo{
					Name:      "test",
					Namespace: "woocoo",
					Host:      "127.0.0.1",
					Port:      8080,
					Metadata: map[string]string{
						"version": "1.0.0",
					},
				},
			}, wantErr: false,
		},
		{
			name: "onlyUnregister",
			fields: fields{opts: Options{TTL: time.Second * 600}, registerContext: func() *RegisterContext {
				var err error
				ctx := &RegisterContext{}
				ctx.providerAPI, err = api.NewProviderAPIByFile(testdata.Path("etc/polaris/polaris.yaml"))
				assert.NoError(t, err)
				return ctx
			}()},
			args: args{
				serviceInfo: &registry.ServiceInfo{
					Name:      "test",
					Namespace: "woocoo",
					Host:      "127.0.0.1",
					Port:      8080,
					Metadata: map[string]string{
						"version": "1.0.0",
					},
				},
				testType: 1,
			}, wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Registry{
				opts:            tt.fields.opts,
				registerContext: tt.fields.registerContext,
			}
			if tt.args.testType == 0 {
				if err := r.Register(tt.args.serviceInfo); (err != nil) != tt.wantErr {
					t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
			if err := r.Unregister(tt.args.serviceInfo); (err != nil) != tt.wantErr {
				t.Errorf("Unregister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
