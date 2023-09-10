package conf

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/test/testdata"
	"os"
	"testing"
)

func TestGetIP(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		ip := GetIP(false)
		assert.NotEqual(t, ip, "error")
		assert.NotContains(t, ip, ":")
	})

	t.Run("ipv6", func(t *testing.T) {
		ip := GetIP(true)
		assert.NotEqual(t, ip, "error")
	})
}

func TestTLS(t *testing.T) {
	tests := []struct {
		name    string
		cnf     *Configuration
		check   func(t *testing.T, tls *TLS)
		wantErr bool
	}{
		{
			name: "normal",
			cnf: NewFromStringMap(map[string]any{
				"ca":   "x509/tls-ca-chain.pem",
				"cert": "x509/server.crt",
				"key":  "x509/server.key",
			}),
			check: func(t *testing.T, tls *TLS) {
				assert.Equal(t, testdata.Path("x509/server.crt"), tls.Cert)
				assert.Equal(t, testdata.Path("x509/server.key"), tls.Key)
			},
		},
		{
			name: "from-bytes",
			cnf: NewFromStringMap(map[string]any{
				"cert": testdata.FileBytes("x509/server.crt"),
				"key":  testdata.FileBytes("x509/server.key"),
			}),
			check: func(t *testing.T, tls *TLS) {
				assert.IsType(t, "", tls.Cert)
				assert.IsType(t, "", tls.Key)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cnf.SetBaseDir(testdata.BaseDir())
			tls := NewTLS(tt.cnf)
			tt.check(t, tls)
			c, err := tls.BuildTlsConfig()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}

func Test_tryLoadEnvFromFiles(t *testing.T) {
	type args struct {
		scan string
		mod  string
	}
	tests := []struct {
		name  string
		args  args
		check func()
		panic bool
	}{
		{
			name: "stat error",
			args: args{
				scan: testdata.Path("etc"),
				mod:  "test",
			},
			check: func() {
				assert.Equal(t, "bartest", os.Getenv("FOO"))
				assert.Equal(t, "1", os.Getenv("INT"))
			},
		},
		{
			name: "stat error",
			args: args{
				scan: "/error",
				mod:  "",
			},
			panic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				assert.Panics(t, func() {
					TryLoadEnvFromFile(tt.args.scan, tt.args.mod)
				})
				return
			}
			TryLoadEnvFromFile(tt.args.scan, tt.args.mod)

			if tt.check != nil {
				tt.check()
			}
		})
	}
}
