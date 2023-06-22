package httpx

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsingsun/woocoo/pkg/conf"
	"golang.org/x/net/http/httpproxy"
)

// ClientConfig configures an HTTP client.
type ClientConfig struct {
	TransportConfig
	// The HTTP basic authentication credentials for the targets.
	BasicAuth *BasicAuth `yaml:"basic_auth,omitempty" json:"basic_auth,omitempty"`
	// The HTTP authorization credentials for the targets.
	Authorization *Authorization `yaml:"authorization,omitempty" json:"authorization,omitempty"`
	// The OAuth2 client credentials used to fetch a token for the targets.
	OAuth2 *OAuth2 `yaml:"oauth2,omitempty" json:"oauth2,omitempty"`
}

// Authorization contains HTTP authorization credentials.
type Authorization struct {
	Type        string `yaml:"type,omitempty" json:"type,omitempty"`
	Credentials string `yaml:"credentials,omitempty" json:"credentials,omitempty"`
}

// BasicAuth contains basic HTTP authentication credentials.
type BasicAuth struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// OAuth2 is the oauth2 client configuration.
type OAuth2 struct {
	ClientID       string            `yaml:"clientId" json:"clientId"`
	ClientSecret   string            `yaml:"clientSecret" json:"clientSecret"`
	Scopes         []string          `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	TokenURL       string            `yaml:"tokenUrl" json:"tokenUrl"`
	EndpointParams map[string]string `yaml:"endpointParams,omitempty" json:"endpointParams,omitempty"`
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
	if c.OAuth2 != nil {
		if c.OAuth2.ClientID == "" {
			return fmt.Errorf("oauth2 clientId must be configured")
		}
		if c.OAuth2.ClientSecret == "" {
			return fmt.Errorf("either oauth2 clientSecret must be configured")
		}
		if c.OAuth2.TokenURL == "" {
			return fmt.Errorf("oauth2 tokenUrl must be configured")
		}
	}
	return nil
}

type TransportConfig struct {
	*ProxyConfig `yaml:",inline" json:",inline"`
	// TLSConfig to use to connect to the targets.
	TLS *conf.TLS `yaml:"tls,omitempty"`
}

type ProxyConfig struct {
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

func NewTransport(cfg TransportConfig) (http.RoundTripper, error) {
	ts := http.DefaultTransport.(*http.Transport)
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

func NewClient(base http.RoundTripper, cfg ClientConfig) (c *http.Client, err error) {
	if base == nil {
		base, err = NewTransport(cfg.TransportConfig)
		if err != nil {
			return nil, err
		}
	}
	return &http.Client{Transport: base}, nil
}
