package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
)

type (
	KeyAuthConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper
		// Exclude is a list of http paths to exclude from key auth
		Exclude []string `json:"exclude" yaml:"exclude"`
		// KeyLookupFuncs defines a list of user-defined functions that extract key token from the given context.
		// This is one of the two options to provide a token extractor.
		// The order of precedence is user-defined KeyLookupFuncs, and KeyLookup.
		// You can also provide both if you want.
		KeyLookupFuncs []ValuesExtractor
		// KeyLookup is a string in the form of "<source>:<name>" that is used
		// to extract key from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		// - "form:<name>"
		KeyLookup string `json:"keyLookup" yaml:"keyLookup"`
		// AuthScheme to be used in the Authorization header.
		AuthScheme string `json:"authScheme" yaml:"authScheme"`
		// Validator is a function to validate key token.You can use it to check the token is valid or not,then set
		// the user info to the context.
		Validator KeyAuthValidator
		// ErrorHandler is a function which is executed when an error occurs during the middleware processing.
		ErrorHandler KeyAuthErrorHandler
	}
	// KeyAuthValidator is a function that validates key token and returns
	KeyAuthValidator func(c *gin.Context, keyAuth string) (bool, error)

	// KeyAuthErrorHandler defines a function which is executed for an invalid token.
	KeyAuthErrorHandler func(c *gin.Context, err error) error
)

var (
	defaultKeyAuthConfig = KeyAuthConfig{
		KeyLookup:  "header:X-Api-Key",
		AuthScheme: "",
	}
)

type KeyAuthMiddleware struct {
	opts middlewareOptions
}

func KeyAuth(opts ...MiddlewareOption) *KeyAuthMiddleware {
	md := &KeyAuthMiddleware{}
	md.opts.applyOptions(opts...)
	return md
}

func (mw *KeyAuthMiddleware) Name() string {
	return "keyAuth"
}

// Shutdown nothing to do
func (mw *KeyAuthMiddleware) Shutdown(_ context.Context) error {
	return nil
}

func (mw *KeyAuthMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opts := new(KeyAuthConfig)
	if mw.opts.configFunc != nil {
		opts = mw.opts.configFunc().(*KeyAuthConfig)
	} else {
		*opts = defaultKeyAuthConfig
	}
	if err := cfg.Unmarshal(&opts); err != nil {
		panic(err)
	}
	if opts.Skipper == nil {
		opts.Skipper = func(c *gin.Context) bool {
			path := c.Request.URL.Path
			for _, p := range opts.Exclude {
				if p == path {
					return true
				}
			}
			return false
		}
	}
	if opts.KeyLookup == "" {
		opts.KeyLookup = defaultKeyAuthConfig.KeyLookup
	}
	if opts.AuthScheme == "" {
		opts.AuthScheme = defaultKeyAuthConfig.AuthScheme
	}
	return keyAuthWithOption(opts)
}

func keyAuthWithOption(opts *KeyAuthConfig) gin.HandlerFunc {
	extractors, err := CreateExtractors(opts.KeyLookup, opts.AuthScheme)
	if err != nil {
		panic(err)
	}
	if len(opts.KeyLookupFuncs) > 0 {
		extractors = append(opts.KeyLookupFuncs, extractors...)
	}
	if opts.Validator == nil {
		panic("keyauth middleware validator must be set")
	}

	return func(c *gin.Context) {
		if opts.Skipper(c) {
			c.Next()
			return
		}
		var lastExtractorErr error
		var lastAuthErr error
		for _, extractor := range extractors {
			keys, err := extractor(c)
			if err != nil {
				lastExtractorErr = err
				continue
			}
			for _, key := range keys {
				ok, err := opts.Validator(c, key)
				if err != nil {
					lastAuthErr = err
					continue
				}
				if ok {
					c.Next()
				}
				return
			}
		}
		err := lastAuthErr
		if err == nil {
			err = lastExtractorErr
		}
		if opts.ErrorHandler != nil {
			err = opts.ErrorHandler(c, err)
			if err == nil {
				c.Next()
				return
			}
		}

		if err != nil {
			c.Error(err) //nolint:errcheck
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}