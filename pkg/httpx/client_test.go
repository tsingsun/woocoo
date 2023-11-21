package httpx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/tsingsun/woocoo/pkg/cache/redisc"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

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

func newTestServer(handler func(w http.ResponseWriter, r *http.Request), usetls bool) (*httptest.Server, error) {
	testServer := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	if usetls {
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
	} else {
		testServer.Start()
	}

	return testServer, nil
}

type customerTokenSource struct {
}

func (c *customerTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: "test",
	}, nil
}

type customerTokenStorage struct {
	t       *oauth2.Token
	needErr bool
}

func (c *customerTokenStorage) Token() (*oauth2.Token, error) {
	if c.needErr {
		return nil, fmt.Errorf("test error")
	}
	return c.t, nil
}

func (c *customerTokenStorage) SetToken(t *oauth2.Token) error {
	c.t = t
	return nil
}

func TestNewClient_TLS(t *testing.T) {
	expectedRes := "Hello"
	type args struct {
		cfg *ClientConfig
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
				cfg: &ClientConfig{
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
				cfg: &ClientConfig{
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
		{
			name: "with empty options",
			args: args{
				cfg: func() *ClientConfig {
					c, err := NewClientConfig(conf.New())
					require.NoError(t, err)
					return c
				}(),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := NewClient(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			usetls := tt.args.cfg.TLS != nil
			ts, err := newTestServer(tt.handler, usetls)
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
				OAuth2: &OAuth2Config{
					Config: oauth2.Config{},
				},
			},
			wantErr: true,
		},
		{
			name: "missing oauth2 clientSecret",
			cfg: ClientConfig{
				OAuth2: &OAuth2Config{
					Config: oauth2.Config{
						ClientID: "id1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing oauth2 tokenURL",
			cfg: ClientConfig{
				OAuth2: &OAuth2Config{
					Config: oauth2.Config{
						ClientID:     "id1",
						ClientSecret: "secret1",
					}},
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

func TestNewClient_Option(t *testing.T) {
	expectedRes := "Hello"
	type args struct {
		cnf  *conf.Configuration
		opts []Option
		cfg  *ClientConfig
	}
	tests := []struct {
		name    string
		args    args
		handler func(w http.ResponseWriter, r *http.Request)
		wantErr bool
	}{
		{
			name: "with empty options",
			args: args{
				cnf: conf.New(),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
		{
			name: "with basic transport",
			args: args{
				cnf: conf.New(),
				opts: []Option{
					WithBase(&http.Transport{}),
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
		{
			name: "with cnf",
			args: args{
				cnf: conf.NewFromStringMap(map[string]any{
					"basicAuth": map[string]any{
						"username": "user",
						"password": "pass",
					},
				}),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				username, password, ok := r.BasicAuth()
				require.True(t, ok)
				require.Equal(t, "user", username)
				require.Equal(t, "pass", password)
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
		{
			name: "with middleware",
			args: args{
				cnf: conf.New(),
				opts: []Option{
					WithMiddleware(func(next http.RoundTripper) http.RoundTripper {
						return internalRoundTripper(func(r *http.Request) (*http.Response, error) {
							r.Header.Set("X-Test", "test")
							return next.RoundTrip(r)
						})
					}, func(next http.RoundTripper) http.RoundTripper {
						return internalRoundTripper(func(r *http.Request) (*http.Response, error) {
							r.Header.Set("X-Test2", "test2")
							return next.RoundTrip(r)
						})
					}),
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "test", r.Header.Get("X-Test"))
				require.Equal(t, "test2", r.Header.Get("X-Test2"))
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
		{
			name: "tokensource",
			args: args{
				cnf: conf.NewFromStringMap(map[string]any{
					"oauth2": map[string]any{
						"clientID":     "id1",
						"clientSecret": "secret1",
						"endpoint": map[string]any{
							"tokenUrl": "http://localhost:8080/token",
						},
					},
				}),
				opts: []Option{
					WithTokenSource(&customerTokenSource{}), WithTokenStorage(&customerTokenStorage{}),
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, expectedRes)
			},
		},
		{
			name: "tokensource with error",
			args: args{
				cnf: conf.NewFromStringMap(map[string]any{
					"oauth2": map[string]any{
						"clientSecret": "secret1",
					},
				}),
				opts: []Option{
					WithTokenSource(&customerTokenSource{}),
				},
			},
			wantErr: true,
		},
		{
			name: "cache not found",
			args: args{
				cnf: conf.NewFromStringMap(map[string]any{
					"oauth2": map[string]any{
						"clientID":     "id1",
						"clientSecret": "secret1",
						"endpoint": map[string]any{
							"tokenUrl": "http://localhost:8080/token",
						},
						"storeKey": "miss",
					},
				}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := NewClientConfig(tt.args.cnf, tt.args.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			client, err := gotC.Client(context.Background(), nil)
			require.NoError(t, err)
			usetls := gotC.TLS != nil
			ts, err := newTestServer(tt.handler, usetls)
			require.NoError(t, err)
			defer ts.Close()
			res, err := client.Get(ts.URL)
			require.NoError(t, err)
			bd, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, expectedRes, string(bd))
		})
	}
}

func TestOAuth2(t *testing.T) {
	tokencount := 0
	const wantAuth = "/auth?client_id=client_id&response_type=code&scope=scope1+scope2&state=state"
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/auth":
			w.Write([]byte(wantAuth))
		case "/token":
			tokencount++
			w.Header().Set("Content-Type", "application/json")
			d, err := json.Marshal(map[string]string{
				"access_token": "90d64460d14870c08c81352a05dedd3465940a7c",
				"expires_in":   "11", // defaultExpiryDelta = 10 * time.Second, so set 11 seconds and sleep 1 second
				"scope":        "user",
				"token_type":   "bearer",
			})
			require.NoError(t, err)
			w.Write(d)
		case "/get":
			require.Equal(t, r.Header.Get("Authorization"), "Bearer 90d64460d14870c08c81352a05dedd3465940a7c")
			_, _ = fmt.Fprint(w, "Hello")
		default:
			t.Errorf("Unexpected exchange request URL %q", r.URL)
		}
	}))
	ts.Start()
	defer ts.Close()
	t.Run("token", func(t *testing.T) {
		tc := ClientConfig{
			Timeout: 2 * time.Second,
			OAuth2: &OAuth2Config{
				Config: oauth2.Config{
					ClientID: "client",
					Endpoint: oauth2.Endpoint{
						TokenURL: ts.URL + "/token",
					},
				},
			},
		}

		client, err := tc.Client(context.Background(), nil)
		require.NoError(t, err)
		res, err := client.Get(ts.URL + "/get")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		_, err = client.Get(ts.URL + "/get")
		require.NoError(t, err)
		time.Sleep(1 * time.Second)
		_, err = client.Get(ts.URL + "/get")
		require.NoError(t, err)
		assert.Equal(t, 2, tokencount)
	})
	t.Run("with cache", func(t *testing.T) {
		mr := miniredis.RunT(t)
		_, err := redisc.New(conf.NewFromStringMap(map[string]any{
			"driverName": "redis",
			"addrs":      []string{mr.Addr()},
		}))
		require.NoError(t, err)
		cnf := conf.New(conf.WithLocalPath(testdata.Path("httpx/all.yaml"))).Load().Sub("oauth-with-cache")
		cnf.Parser().Set("oauth2.endpoint.tokenURL", ts.URL+cnf.String("oauth2.endpoint.tokenURL"))
		cnf.Parser().Set("oauth2.endpoint.authURL", ts.URL+cnf.String("oauth2.endpoint.authURL"))
		cfg, err := NewClientConfig(cnf)
		require.NoError(t, err)
		t.Run("client", func(t *testing.T) {
			tokencount = 0
			client, err := cfg.Client(context.Background(), nil)
			require.NoError(t, err)
			res, err := client.Get(ts.URL + "/get")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, res.StatusCode)
			_, err = client.Get(ts.URL + "/get")
			require.NoError(t, err)
			time.Sleep(1 * time.Second)
			_, err = client.Get(ts.URL + "/get")
			require.NoError(t, err)
			assert.Equal(t, 2, tokencount)

			v, err := mr.Get(tokenKey(cfg.OAuth2))
			require.NoError(t, err)
			assert.NotNil(t, v)
		})
		t.Run("exchange", func(t *testing.T) {
			tokencount = 0
			codeURL := cfg.OAuth2.AuthCodeURL("state")
			require.Contains(t, codeURL, wantAuth)
			token, err := cfg.Exchange(context.Background(), "code")
			require.NoError(t, err)
			client, err := cfg.Client(context.Background(), token)
			require.NoError(t, err)
			res, err := client.Get(ts.URL + "/get")
			require.NoError(t, err)
			bd, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Contains(t, string(bd), "Hello")
			assert.Equal(t, 1, tokencount)
		})
	})
}
