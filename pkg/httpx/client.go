package httpx

import (
	"context"
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/tsingsun/woocoo/pkg/cache"
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
	// TokenStorage is an interface to store and retrieve oauth2 token
	TokenStorage interface {
		Token() (*oauth2.Token, error)
		SetToken(*oauth2.Token) error
	}

	// ClientConfig is for an extension http.Client. It can be used to configure a client with configuration.
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

	TransportConfig struct {
		*ProxyConfig `yaml:",inline" json:",inline"`
		// TLSConfig to use to connect to the targets.
		TLS *conf.TLS `yaml:"tls,omitempty" json:"tls,omitempty"`
	}
)

// NewClientConfig creates a new ClientConfig by options.
func NewClientConfig(cnf *conf.Configuration, opts ...Option) (cfg *ClientConfig, err error) {
	cfg = &ClientConfig{}
	if err = cnf.Unmarshal(cfg); err != nil {
		return
	}
	if err = cfg.Validate(); err != nil {
		return
	}

	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.BasicAuth != nil {
		cfg.base = chain(cfg.base, BaseAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password))
	}
	if cfg.OAuth2 != nil && cfg.OAuth2.StoreKey != "" {
		storage, err := newCacheTokenStorage(cfg)
		if err != nil {
			return cfg, err
		}
		cfg.OAuth2.storage = storage
	}
	return cfg, nil
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
		if c.OAuth2.Endpoint.TokenURL == "" {
			return fmt.Errorf("oauth2 tokenUrl must be configured")
		}
	}
	return nil
}

// Exchange converts an authorization code into a token if you use oauth2 config.
func (c *ClientConfig) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if c.OAuth2 == nil {
		return nil, fmt.Errorf("oauth2 config not found")
	}
	tk, err := c.OAuth2.Config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}
	if c.OAuth2.storage != nil {
		if err := c.OAuth2.storage.SetToken(tk); err != nil {
			return nil, err
		}
	}
	return tk, nil
}

// Client returns an HTTP client using the provided token.
func (c *ClientConfig) Client(ctx context.Context, t *oauth2.Token) (*http.Client, error) {
	if t != nil && c.OAuth2 == nil {
		return nil, fmt.Errorf("oauth2 config not found")
	}
	if t != nil && c.OAuth2 != nil && c.OAuth2.ts == nil {
		c.OAuth2.ts = c.OAuth2.Config.TokenSource(ctx, t)
	}
	return NewClient(c)
}

// TokenSource returns a default token source base on clientcredentials.Config. it called in NewClient
func (c *ClientConfig) TokenSource(ctx context.Context) oauth2.TokenSource {
	config := &clientcredentials.Config{
		ClientID:       c.OAuth2.ClientID,
		ClientSecret:   c.OAuth2.ClientSecret,
		Scopes:         c.OAuth2.Scopes,
		TokenURL:       c.OAuth2.Endpoint.TokenURL,
		EndpointParams: c.OAuth2.EndpointParams,
	}
	hc := &http.Client{Transport: c.base, Timeout: c.Timeout}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, hc)
	base := config.TokenSource(context.WithValue(ctx, oauth2.HTTPClient, hc))
	return &TokenSource{
		storage: c.OAuth2.storage,
		base:    base,
	}
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

// OAuth2Config is a wrapper around oauth2.Config that allows for custom.
type OAuth2Config struct {
	oauth2.Config `yaml:",inline" json:",inline"`
	// StoreKey is the name of the cache driver which is used to store token.
	// Default is empty. If StoreKey is empty, the token will not be cached.
	StoreKey       string `json:"storeKey" yaml:"storeKey"`
	EndpointParams url.Values

	ts      oauth2.TokenSource
	storage TokenStorage
}

// SetOAuthStorage set TokenStorage to OAuth2Config
func (oa *OAuth2Config) SetOAuthStorage(ts TokenStorage) error {
	oa.storage = ts
	return nil
}

type TokenSource struct {
	storage TokenStorage
	base    oauth2.TokenSource
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	if t.storage == nil {
		return t.base.Token()
	}
	if token, err := t.storage.Token(); err == nil && token.Valid() {
		return token, err
	}
	token, err := t.base.Token()
	if err != nil {
		return token, err
	}
	if err := t.storage.SetToken(token); err != nil {
		return nil, err
	}
	return token, nil
}

// cacheTokenStorage is an implementation of TokenStorage that stores the token in a cache.Cache.
type cacheTokenStorage struct {
	config *ClientConfig
	cache  cache.Cache

	tokenCacheKey string
}

// cnf is the client configuration
func newCacheTokenStorage(cfg *ClientConfig) (*cacheTokenStorage, error) {
	dc, err := buildCache(cfg.OAuth2.StoreKey)
	if err != nil {
		return nil, err
	}
	return &cacheTokenStorage{
		config:        cfg,
		cache:         dc,
		tokenCacheKey: tokenKey(cfg.OAuth2),
	}, nil
}

func tokenKey(config *OAuth2Config) string {
	// hash cfg
	hash, err := hashstructure.Hash(config, hashstructure.FormatV2, nil)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s:token:%d", config.ClientID, hash)
}

func buildCache(key string) (cache.Cache, error) {
	return cache.GetCache(key)
}

func (c *cacheTokenStorage) Token() (*oauth2.Token, error) {
	if c.cache == nil {
		return nil, nil
	}
	t := &oauth2.Token{}
	err := c.cache.Get(context.Background(), c.tokenCacheKey, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (c *cacheTokenStorage) SetToken(t *oauth2.Token) error {
	if !t.Valid() {
		return fmt.Errorf("invalid token")
	}
	return c.cache.Set(context.Background(), c.tokenCacheKey, t, cache.WithTTL(time.Until(t.Expiry)))
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
func NewClient(cfg *ClientConfig) (c *http.Client, err error) {
	if cfg.base == nil {
		cfg.base, err = NewTransport(cfg.TransportConfig)
		if err != nil {
			return nil, err
		}
	}
	c = &http.Client{Transport: cfg.base, Timeout: cfg.Timeout}
	if cfg.OAuth2 != nil {
		if cfg.OAuth2.ts != nil {
			c.Transport = &oauth2.Transport{
				Base:   c.Transport,
				Source: cfg.OAuth2.ts,
			}
			return
		} else {
			c.Transport = &oauth2.Transport{
				Base:   c.Transport,
				Source: cfg.TokenSource(context.Background()),
			}
			return
		}
	}
	return
}
