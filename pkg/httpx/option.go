package httpx

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"golang.org/x/oauth2"
	"net/http"
)

type Option func(c *ClientConfig)

// internalRoundTripper is a holder function to make the process of
// creating middleware a bit easier without requiring the consumer to
// implement the RoundTripper interface.
type internalRoundTripper func(*http.Request) (*http.Response, error)

func (rt internalRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt(req)
}

// Middleware is our middleware creation functionality.
type Middleware func(http.RoundTripper) http.RoundTripper

// chain is a handy function to wrap a base RoundTripper (optional)
// with the middlewares.
func chain(rt http.RoundTripper, middlewares ...Middleware) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}

	for _, m := range middlewares {
		rt = m(rt)
	}

	return rt
}

// WithConfiguration init from configuration
func WithConfiguration(cnf *conf.Configuration) Option {
	return func(c *ClientConfig) {
		if err := cnf.Unmarshal(c); err != nil {
			panic(err)
		}
	}
}

func WithBase(base http.RoundTripper) Option {
	return func(c *ClientConfig) {
		c.base = base
	}
}

func WithMiddleware(middleware ...Middleware) Option {
	return func(c *ClientConfig) {
		c.base = chain(c.base, middleware...)
	}
}

// BaseAuth is a middleware that adds basic auth to the request.
func BaseAuth(username, password string) Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return internalRoundTripper(func(req *http.Request) (*http.Response, error) {
			req.SetBasicAuth(username, password)
			return rt.RoundTrip(req)
		})
	}
}

// TokenSource is a middleware that handle oauth2 request.
func TokenSource(source oauth2.TokenSource) Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{
			Base:   rt,
			Source: source,
		}
	}
}
