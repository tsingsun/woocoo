package httpx

import (
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

func WithBase(base http.RoundTripper) Option {
	return func(c *ClientConfig) {
		c.base = base
	}
}

// WithTokenSource set oauth2 token source after oauth2 config initialized
func WithTokenSource(source oauth2.TokenSource) Option {
	return func(c *ClientConfig) {
		c.OAuth2.ts = source
	}
}

// WithTokenStorage set oauth2 token storage after oauth2 config initialized
func WithTokenStorage(storage TokenStorage) Option {
	return func(c *ClientConfig) {
		c.OAuth2.storage = storage
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
