package httpx

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tsingsun/woocoo/pkg/conf"
	"golang.org/x/net/http/httpproxy"
)

type (
	// ClientConfig configures an HTTP client.
	ClientConfig struct {
		TransportConfig
		Timeout time.Duration `yaml:"timeout" json:"timeout"`
		// The HTTP basic authentication credentials for the targets.
		BasicAuth *BasicAuth `yaml:"basicAuth,omitempty" json:"basicAuth,omitempty"`
		// The HTTP authorization credentials for the targets.
		Authorization *Authorization `yaml:"authorization,omitempty" json:"authorization,omitempty"`
		// The OAuth2 client credentials used to fetch a token for the targets.
		OAuth2 *OAuth2Config `yaml:"oauth2,omitempty" json:"oauth2,omitempty"`

		base http.RoundTripper
	}

	// Authorization contains HTTP authorization credentials.
	Authorization struct {
		Type        string `yaml:"type,omitempty" json:"type,omitempty"`
		Credentials string `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	}

	// BasicAuth contains basic HTTP authentication credentials.
	BasicAuth struct {
		Username string `yaml:"username" json:"username"`
		Password string `yaml:"password,omitempty" json:"password,omitempty"`
	}

	OAuth2Config struct {
		*oauth2.Config
		EndpointParams url.Values
	}

	TransportConfig struct {
		*ProxyConfig `yaml:",inline" json:",inline"`
		// TLSConfig to use to connect to the targets.
		TLS *conf.TLS `yaml:"tls,omitempty"`
	}

	ProxyConfig struct {
		// HTTP proxy server to use to connect to the targets.
		ProxyURL string `yaml:"proxyUrl,omitempty" json:"proxyUrl,omitempty"`
		// NoProxy contains addresses that should not use a proxy.
		NoProxy string `yaml:"noProxy,omitempty" json:"noProxy,omitempty"`
		// ProxyConnectHeader optionally specifies headers to send to
		// proxies during CONNECT requests. Assume that at least _some_ of
		// these headers are going to contain secrets and use Secret as the
		// value type instead of string.
		ProxyConnectHeader http.Header `yaml:"proxyConnectHeader,omitempty" json:"proxyConnectHeader,omitempty"`
	}
)

// NewClientConfig creates a new ClientConfig by options.
func NewClientConfig(opts ...Option) ClientConfig {
	cfg := ClientConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.BasicAuth != nil {
		cfg.base = chain(cfg.base, BaseAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password))
	}
	return cfg
}

func (c *ClientConfig) Validate() error {
	if c.Authorization != nil {
		if c.Authorization.Type == "" {
			c.Authorization.Type = "Bearer"
		}
		switch strings.ToLower(c.Authorization.Type) {
		case "bearer", "basic_auth":
		default:
			return fmt.Errorf("authorization type %q no support", c.Authorization.Type)
		}
	}
	if c.OAuth2 != nil && c.OAuth2.Config != nil {
		if c.OAuth2.ClientID == "" {
			return fmt.Errorf("oauth2 clientId must be configured")
		}
		if c.OAuth2.ClientSecret == "" {
			return fmt.Errorf("either oauth2 clientSecret must be configured")
		}
		if c.OAuth2.Endpoint.TokenURL == "" {
			return fmt.Errorf("oauth2 tokenUrl must be configured")
		}
	}
	return nil
}

func (p ProxyConfig) Validate() error {
	if p.ProxyURL != "" {
		if _, err := url.ParseRequestURI(p.ProxyURL); err != nil {
			return fmt.Errorf("proxyUrl %q is invalid: %w", p.ProxyURL, err)
		}
	}
	return nil
}

func (p ProxyConfig) ProxyFunc() func(req *http.Request) (*url.URL, error) {
	proxy := &httpproxy.Config{
		HTTPProxy:  p.ProxyURL,
		HTTPSProxy: p.ProxyURL,
		NoProxy:    p.NoProxy,
	}
	fn := proxy.ProxyFunc()
	return func(req *http.Request) (*url.URL, error) {
		return fn(req.URL)
	}
}

// NewTransport creates a new HTTP transport base on TransportConfig and http.DefaultTransport.
func NewTransport(cfg TransportConfig) (http.RoundTripper, error) {
	df := http.DefaultTransport.(*http.Transport)
	ts := &http.Transport{
		Proxy:                 df.Proxy,
		DialContext:           df.DialContext,
		MaxIdleConns:          df.MaxIdleConns,
		MaxIdleConnsPerHost:   df.MaxIdleConnsPerHost,
		DisableKeepAlives:     df.DisableKeepAlives,
		DisableCompression:    df.DisableCompression,
		IdleConnTimeout:       df.IdleConnTimeout,
		TLSHandshakeTimeout:   df.TLSHandshakeTimeout,
		ExpectContinueTimeout: df.ExpectContinueTimeout,
	}
	if cfg.TLS != nil {
		t, err := cfg.TLS.BuildTlsConfig()
		if err != nil {
			return nil, err
		}
		ts.TLSClientConfig = t
	}
	if cfg.ProxyConfig != nil {
		ts.Proxy = cfg.ProxyConfig.ProxyFunc()
		ts.ProxyConnectHeader = cfg.ProxyConfig.ProxyConnectHeader
	}
	return ts, nil
}

// NewClient creates a new HTTP client.
//
// OAuth2 Client from Configuration is use client credentials flow.You can use TokenSource to custom Source.
func NewClient(cfg ClientConfig) (c *http.Client, err error) {
	if cfg.base == nil {
		cfg.base, err = NewTransport(cfg.TransportConfig)
		if err != nil {
			return nil, err
		}
	}
	c = &http.Client{Transport: cfg.base, Timeout: cfg.Timeout}
	if cfg.OAuth2 != nil {
		config := &clientcredentials.Config{
			ClientID:       cfg.OAuth2.ClientID,
			ClientSecret:   cfg.OAuth2.ClientSecret,
			Scopes:         cfg.OAuth2.Scopes,
			TokenURL:       cfg.OAuth2.Endpoint.TokenURL,
			EndpointParams: cfg.OAuth2.EndpointParams,
		}
		hc := &http.Client{Transport: cfg.base}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hc)
		c.Transport = &oauth2.Transport{
			Base:   hc.Transport,
			Source: config.TokenSource(ctx),
		}

		return
	}
	return
}

// NewClientFromCnf creates a new HTTP client from config.
func NewClientFromCnf(cnf *conf.Configuration) (*http.Client, error) {
	cfg := NewClientConfig(WithConfiguration(cnf))
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return NewClient(cfg)
}
