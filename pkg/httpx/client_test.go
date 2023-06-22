package httpx

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/test/testdata"
)

const (
	TLSCAChain = "x509/tls-ca-chain.pem"
	serverCert = "x509/server.crt"
	serverKey  = "x509/server.key"
	clientCert = "x509/client.crt"
	clientKey  = "x509/client.key"
)

func newTestServer(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, error) {
	testServer := httptest.NewUnstartedServer(http.HandlerFunc(handler))

	tlsCAChain, err := os.ReadFile(testdata.Path(TLSCAChain))
	if err != nil {
		return nil, fmt.Errorf("can't read ca file")
	}
	serverCertificate, err := tls.LoadX509KeyPair(testdata.Path(serverCert), testdata.Path(serverKey))
	if err != nil {
		return nil, fmt.Errorf("can't load X509 key pair %s - %s", serverCert, serverKey)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(tlsCAChain)

	testServer.TLS = &tls.Config{
		Certificates: make([]tls.Certificate, 1),
		RootCAs:      rootCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    rootCAs}
	testServer.TLS.Certificates[0] = serverCertificate

	testServer.StartTLS()

	return testServer, nil
}

func TestNewClient(t *testing.T) {
	expectedRes := "Hello"
	type args struct {
		base http.RoundTripper
		cfg  ClientConfig
	}
	tests := []struct {
		name    string
		args    args
		handler func(w http.ResponseWriter, r *http.Request)
		wantErr bool
	}{
		{
			name: "tls",
			args: args{
				base: nil,
				cfg: ClientConfig{
					TransportConfig: TransportConfig{
						TLS: &conf.TLS{
							CA:                 testdata.Path(TLSCAChain),
							Cert:               testdata.Path(clientCert),
							Key:                testdata.Path(clientKey),
							InsecureSkipVerify: false,
						},
					},
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
		{
			name: "tls with proxy",
			args: args{
				base: nil,
				cfg: ClientConfig{
					TransportConfig: TransportConfig{
						TLS: &conf.TLS{
							CA:                 testdata.Path(TLSCAChain),
							Cert:               testdata.Path(clientCert),
							Key:                testdata.Path(clientKey),
							InsecureSkipVerify: false,
						},
						ProxyConfig: &ProxyConfig{
							ProxyURL: "http://127.0.0.1:80",
						},
					},
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := NewClient(tt.args.base, tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			ts, err := newTestServer(tt.handler)
			require.NoError(t, err)
			defer ts.Close()
			res, err := gotC.Get(ts.URL)
			require.NoError(t, err)
			bd, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, expectedRes, string(bd))
		})
	}
}

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: ClientConfig{
				BasicAuth: &BasicAuth{
					Username: "user1",
					Password: "password1",
				},
			},
			wantErr: false,
		},
		{
			name: "empty auth type",
			cfg: ClientConfig{
				Authorization: &Authorization{
					Type: "",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid auth type",
			cfg: ClientConfig{
				Authorization: &Authorization{
					Type: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "missing oauth2 clientID",
			cfg: ClientConfig{
				OAuth2: &OAuth2{},
			},
			wantErr: true,
		},
		{
			name: "missing oauth2 clientSecret",
			cfg: ClientConfig{
				OAuth2: &OAuth2{
					ClientID: "id1",
				},
			},
			wantErr: true,
		},
		{
			name: "missing oauth2 tokenURL",
			cfg: ClientConfig{
				OAuth2: &OAuth2{
					ClientID:     "id1",
					ClientSecret: "secret1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProxyConfig
		wantErr bool
	}{
		{
			name: "valid proxy url",
			cfg: ProxyConfig{
				ProxyURL: "http://127.0.0.1:8080",
			},
			wantErr: false,
		},
		{
			name: "invalid proxy url",
			cfg: ProxyConfig{
				ProxyURL: "invalidabc",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProxyFunc(t *testing.T) {
	tests := []struct {
		name     string
		proxyURL string
		noProxy  string
		url      string
		want     string
	}{
		{
			name:     "proxy used",
			proxyURL: "http://localhost:8000",
			url:      "http://example.com",
			want:     "http://localhost:8000",
		},
		{
			name:     "no proxy for localhost",
			proxyURL: "http://localhost:8000",
			url:      "http://localhost/foo",
			want:     "",
		},
		{
			name:     "no proxy from noProxy",
			proxyURL: "http://localhost:8000",
			noProxy:  "example.com",
			url:      "http://example.com/foo",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ProxyConfig{
				ProxyURL: tt.proxyURL,
				NoProxy:  tt.noProxy,
			}
			proxyFunc := p.ProxyFunc()
			urlstr, err := url.Parse(tt.url)
			assert.NoError(t, err)
			u, err := proxyFunc(&http.Request{URL: urlstr})
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				assert.Nil(t, u)
			} else {
				assert.Equal(t, tt.want, u.String())
			}
		})
	}
}
