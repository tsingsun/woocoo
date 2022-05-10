package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
)

type (
	JWTOptions struct {
		auth.JWTOptions `json:",squash" yaml:",squash"`

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
}

func JWT() *JWTMiddleware {
	return &JWTMiddleware{}
}

func (mw *JWTMiddleware) Name() string {
	return "jwt"
}

func (mw *JWTMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opts := &JWTOptions{
		JWTOptions: *auth.NewJWT(),
	}
	if err := cfg.Unmarshal(&opts); err != nil {
		panic(err)
	}
	return jwtWithOption(opts)
}

// Shutdown no need to do anything
func (mw *JWTMiddleware) Shutdown() {
}

func jwtWithOption(opts *JWTOptions) gin.HandlerFunc {
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
		var lastExtractorErr error
		var lastTokenErr error
		for _, extractor := range extractors {
			auths, err := extractor(c)
			if err != nil {
				lastExtractorErr = auth.ErrJWTMissing // backwards compatibility: all extraction errors are same (unlike KeyAuth)
				continue
			}
			for _, authStr := range auths {
				token, err := opts.ParseTokenFunc(authStr, c)
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
			opts.ErrorHandler(err)
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"errors": []map[string]interface{}{
					{
						"code":    http.StatusUnauthorized,
						"message": auth.ErrJWTInvalid.Error(),
					},
				},
			})
		}
	}
}
