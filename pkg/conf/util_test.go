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

func TestParseEnv(t *testing.T) {
	// 设置测试环境变量
	t.Setenv("TEST_HOST", "localhost")
	t.Setenv("TEST_PORT", "8080")
	t.Setenv("TEST_DB.HOST", "db.example.com")

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "basic substitution",
			src:  "host: ${TEST_HOST}",
			want: "host: localhost",
		},
		{
			name: "with spaces",
			src:  "host: ${ TEST_HOST }",
			want: "host: localhost",
		},
		{
			name: "multiple vars",
			src:  "addr: ${TEST_HOST}:${TEST_PORT}",
			want: "addr: localhost:8080",
		},
		{
			name: "unset var becomes empty",
			src:  "key: ${UNSET_VAR}",
			want: "key: ",
		},
		{
			name: "default value when unset",
			src:  "key: ${UNSET_VAR:-default_value}",
			want: "key: default_value",
		},
		{
			name: "default value with spaces",
			src:  "key: ${ UNSET_VAR :- default_value }",
			want: "key: default_value",
		},
		{
			name: "env overrides default",
			src:  "host: ${TEST_HOST:-fallback}",
			want: "host: localhost",
		},
		{
			name: "empty default",
			src:  "key: ${UNSET_VAR:-}",
			want: "key: ",
		},
		{
			name: "escaped literal",
			src:  `literal: \${NOT_A_VAR}`,
			want: "literal: ${NOT_A_VAR}",
		},
		{
			name: "dot in var name",
			src:  "db: ${TEST_DB.HOST}",
			want: "db: db.example.com",
		},
		{
			name: "no substitution needed",
			src:  "plain: text",
			want: "plain: text",
		},
		{
			name: "mixed escaped and real",
			src:  `real: ${TEST_HOST}, literal: \${FAKE}`,
			want: "real: localhost, literal: ${FAKE}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseEnv([]byte(tt.src))
			assert.Equal(t, tt.want, string(got))
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
