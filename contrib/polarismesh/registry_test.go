package polarismesh

import (
	"github.com/polarismesh/polaris-go/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/rpc/grpcx/registry"
	"strings"
	"testing"
	"time"
)

func getPolarisContext(t *testing.T, ref string) (ctx api.SDKContext) {
	assert.NotPanics(t, func() {
		var err error
		ctx, err = getPolarisContextFromDriver(ref)
		require.NoError(t, err)
	})
	return
}

func TestRegistry_Apply(t *testing.T) {
	b := []byte(`
grpc:
  server:
    addr: 127.0.0.1:20000
    namespace: /woocoo/service
    version: "1.0"
    ipv6: true
  registry:
    scheme: polaris
    ttl: 600s
    polaris: 
      global:
        serverConnector:
          addresses:
            - 127.0.0.1:8091
`)
	cfg := conf.NewFromBytes(b, conf.WithBaseDir("testdata"))

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
		{name: "in", fields: fields{registerContext: &RegisterContext{}}, args: args{cfg: cfg.Sub("grpc.registry")}},
		{name: "file", fields: fields{registerContext: &RegisterContext{}}, args: args{
			cfg: func() *conf.Configuration {
				c := cfg.Sub("grpc.registry")
				c.Parser().Set("polaris.configFile", "polaris.yaml")
				return c
			}()},
		},
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
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "registerAndUnregister",
			fields: fields{opts: Options{TTL: time.Second * 600}, registerContext: func() *RegisterContext {
				var err error
				ctx := &RegisterContext{}
				ctx.providerAPI, err = api.NewProviderAPIByFile("testdata/polaris.yaml")
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
			}, wantErr: assert.NoError,
		},
		{
			name: "onlyUnregister",
			fields: fields{opts: Options{TTL: time.Second * 600}, registerContext: func() *RegisterContext {
				var err error
				ctx := &RegisterContext{}
				ctx.providerAPI, err = api.NewProviderAPIByFile("testdata/polaris.yaml")
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
			}, wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Registry{
				opts:            tt.fields.opts,
				registerContext: tt.fields.registerContext,
			}
			if tt.args.testType == 0 {
				tt.wantErr(t, r.Register(tt.args.serviceInfo))
			}
			tt.wantErr(t, r.Unregister(tt.args.serviceInfo))

		})
	}
}

func TestRegistry_GetServiceInfos(t *testing.T) {
	var err error
	ctx := &RegisterContext{}
	ctx.providerAPI, err = api.NewProviderAPIByFile("testdata/polaris.yaml")
	assert.NoError(t, err)
	cnf := conf.NewFromBytes([]byte(`
registry:
  name: polarisGetServiceInfos
  scheme: polaris
  ttl: 10s
  polaris:
    global:
      serverConnector:
        addresses:
          - 127.0.0.1:8091
`))
	info := &registry.ServiceInfo{
		Name:      "TestGetServiceInfos",
		Namespace: "woocoo",
		Host:      "127.0.0.1",
		Port:      11111,
		Metadata: map[string]string{
			"version":     "1.0.0",
			"customField": "customValue",
		},
	}
	drv, ok := registry.GetRegistry(scheme)
	require.True(t, ok)
	r, err := drv.(*Driver).CreateRegistry(cnf.Sub("registry"))
	require.NoError(t, err)
	require.NoError(t, r.Register(info))
	r, err = drv.GetRegistry("noExist")
	assert.Error(t, err)
	r, err = drv.GetRegistry("polarisGetServiceInfos")
	require.NoError(t, err)
	time.Sleep(time.Second * 2)
	infos, err := r.GetServiceInfos(strings.Join([]string{info.Namespace, info.Name}, "/"))
	require.NoError(t, err)
	require.Equal(t, 1, len(infos))

	gotInfo := infos[0]
	assert.Equal(t, info.Name, gotInfo.Name)
	assert.Equal(t, info.Namespace, gotInfo.Namespace)
	assert.Equal(t, info.Host, gotInfo.Host)
	assert.Equal(t, info.Port, gotInfo.Port)
	assert.Equal(t, info.Metadata["version"], gotInfo.Metadata["version"])
	assert.Equal(t, info.Metadata["customField"], gotInfo.Metadata["customField"])

	assert.NoError(t, r.Unregister(info), "check GetServiceInfos release consumer api influence.")

	infos, err = r.GetServiceInfos(info.Name)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(infos), "no specify namespace:use default")
}
