package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
)

type (
	JWTConfig struct {
		auth.JWTOptions `json:",squash" yaml:",squash"`
		Skipper         Skipper
		// Exclude is a list of http paths to exclude from JWT auth
		Exclude []string `json:"exclude" yaml:"exclude"`
		// TokenLookupFuncs defines a list of user-defined functions that extract JWT token from the given context.
		// This is one of the two options to provide a token extractor.
		// The order of precedence is user-defined TokenLookupFuncs, and TokenLookup.
		// You can also provide both if you want.
		TokenLookupFuncs []ValuesExtractor

		// SuccessHandler defines a function which is executed for a valid token before middleware chain continues with next
		// middleware or handler.
		SuccessHandler JWTSuccessHandler

		// ErrorHandler defines a function which is executed for an invalid token.
		// It may be used to define a custom JWT error.
		ErrorHandler JWTErrorHandler

		// ErrorHandlerWithContext is almost identical to ErrorHandler, but it's passed the current context.
		ErrorHandlerWithContext JWTErrorHandlerWithContext
	}

	// JWTSuccessHandler defines a function which is executed for a valid token.
	JWTSuccessHandler func(c *gin.Context)

	// JWTErrorHandler defines a function which is executed for an invalid token.
	JWTErrorHandler func(err error) error

	// JWTErrorHandlerWithContext is almost identical to JWTErrorHandler, but it's passed the current context.
	JWTErrorHandlerWithContext func(err error, c *gin.Context) error
)

// JWTMiddleware provides a Json-Web-Token authentication implementation. On failure, a 401 HTTP response
// is returned. On success, the wrapped middleware is called, and the userID is made available as
// c.Get("userID").(string).
// Users can get a token by posting a json request to LoginHandler. The token then needs to be passed in
// the Authentication header. Example: Authorization:Bearer XXX_TOKEN_XXX
type JWTMiddleware struct {
	opts middlewareOptions
}

func JWT(opts ...MiddlewareOption) *JWTMiddleware {
	md := &JWTMiddleware{}
	md.opts.applyOptions(opts...)
	return md
}

func (mw *JWTMiddleware) Name() string {
	return "jwt"
}

func (mw *JWTMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	var opts *JWTConfig
	if mw.opts.configFunc != nil {
		opts = mw.opts.configFunc().(*JWTConfig)
	} else {
		opts = &JWTConfig{
			JWTOptions: *auth.NewJWT(),
		}
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
	return jwtWithOption(opts)
}

// Shutdown no need to do anything
func (mw *JWTMiddleware) Shutdown() {
}

func jwtWithOption(opts *JWTConfig) gin.HandlerFunc {
	if err := opts.Apply(); err != nil {
		panic(err)
	}
	extractors, err := CreateExtractors(opts.TokenLookup, opts.AuthScheme)
	if err != nil {
		panic(err)
	}
	if len(opts.TokenLookupFuncs) > 0 {
		extractors = append(opts.TokenLookupFuncs, extractors...)
	}
	return func(c *gin.Context) {
		if opts.Skipper(c) {
			c.Next()
			return
		}
		var lastExtractorErr error
		var lastTokenErr error
		for _, extractor := range extractors {
			auths, err := extractor(c)
			if err != nil {
				lastExtractorErr = auth.ErrJWTMissing // backwards compatibility: all extraction errors are same (unlike KeyAuth)
				continue
			}
			for _, authStr := range auths {
				token, err := opts.ParseTokenFunc(c, authStr)
				if err != nil {
					lastTokenErr = err
					continue
				}
				// Store user information from token into context.
				c.Set(opts.ContextKey, token)
				if opts.SuccessHandler != nil {
					opts.SuccessHandler(c)
				}
				c.Next()
				return
			}
		}
		// we are here only when we did not successfully extract or parse any of the tokens
		err := lastTokenErr
		if err == nil { // prioritize token errors over extracting errors
			err = lastExtractorErr
		}
		if opts.ErrorHandler != nil {
			opts.ErrorHandler(err) //nolint:errcheck
			return
		}
		if opts.ErrorHandlerWithContext != nil {
			tmpErr := opts.ErrorHandlerWithContext(err, c)
			if opts.ContinueOnIgnoredError && tmpErr == nil {
				c.Next()
				return
			}
		}

		// backwards compatible errors codes
		if lastTokenErr != nil {
			c.JSON(http.StatusUnauthorized, FormatResponseError(http.StatusUnauthorized, lastTokenErr))
		}
	}
}
