package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/tsingsun/woocoo/pkg/auth"
	"github.com/tsingsun/woocoo/pkg/cache"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"net/http"
)

type (
	JWTConfig struct {
		auth.JWTOptions `json:",inline" yaml:",inline"`
		Skipper         Skipper
		// Exclude is a list of http paths to exclude from JWT auth
		//
		// path format must same as url.URL.Path started with "/" and ended with "/"
		Exclude []string `json:"exclude" yaml:"exclude"`
		// TokenLookupFuncs defines a list of user-defined functions that extract JWT token from the given context.
		// This is one of the two options to provide a token extractor.
		// The order of precedence is user-defined TokenLookupFuncs, and TokenLookup.
		// You can also provide both if you want.
		TokenLookupFuncs []ValuesExtractor
		// LogoutHandler defines a function which is executed for user logout system.It clear something like cache.
		LogoutHandler func(*gin.Context)
		// ErrorHandler defines a function which is executed for an invalid token.
		// It may be used to define a custom JWT error and abort the request.like use:
		//  c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		ErrorHandler func(c *gin.Context, err error) error
		// TokenStoreKey is the name of the cache driver,default is "redis".
		// When this option is used, requirements : token cache KEY that uses the JWT ID.
		TokenStoreKey string `json:"tokenStoreKey" yaml:"TokenStoreKey"`
		// WithPrincipalContext defines a function which is Principal creator and store principal in context.
		//
		// Use GeneratePrincipal by default. You can use your own function to create principal.
		WithPrincipalContext func(c *gin.Context, token *jwt.Token) error
	}
)

// JWTMiddleware provides a Json-Web-Token authentication implementation. On failure, a 401 HTTP response
// is returned. On success, the wrapped middleware is called, and the userID is made available as
// c.Get("userID").(string).
// Users can get a token by posting a json request to LoginHandler. The token then needs to be passed in
// the Authentication header. Example: Authorization:Bearer XXX_TOKEN_XXX
type JWTMiddleware struct {
	opts   middlewareOptions
	Config *JWTConfig
	// TokenStore is the cache for store token key.
	TokenStore cache.Cache
}

func JWT(opts ...MiddlewareOption) *JWTMiddleware {
	md := &JWTMiddleware{}
	md.opts.applyOptions(opts...)
	return md
}

func (mw *JWTMiddleware) Name() string {
	return "jwt"
}

func (mw *JWTMiddleware) build(cfg *conf.Configuration) {
	var opts *JWTConfig
	if mw.opts.configFunc != nil {
		opts = mw.opts.configFunc().(*JWTConfig)
	} else {
		opts = &JWTConfig{
			JWTOptions: *auth.NewJWTOptions(),
		}
	}
	if err := cfg.Unmarshal(&opts); err != nil {
		panic(err)
	}
	if err := opts.Init(); err != nil {
		panic(err)
	}

	if opts.Skipper == nil {
		opts.Skipper = func(c *gin.Context) bool {
			return PathSkip(opts.Exclude, c.Request.URL)
		}
	}
	if opts.WithPrincipalContext == nil {
		opts.WithPrincipalContext = func(c *gin.Context, token *jwt.Token) error {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return auth.ErrJWTClaims
			}
			prpl := security.NewGenericPrincipalByClaims(claims)
			c.Request = c.Request.WithContext(security.WithContext(c.Request.Context(), prpl))
			return nil
		}
	}
	mw.Config = opts

	if opts.TokenStoreKey != "" {
		mw.TokenStore = cache.GetCache(opts.TokenStoreKey)
	}

	if opts.LogoutHandler == nil {
		opts.LogoutHandler = func(c *gin.Context) {
			gp := security.GenericPrincipalFromContext(c)
			cl := gp.Identity().Claims()
			jti, ok := opts.GetTokenIDFunc(&jwt.Token{Claims: cl})
			if ok && mw.TokenStore != nil {
				if err := mw.TokenStore.Del(c, jti); err != nil {
					c.Error(err) // nolint: errcheck
				}
			}
		}
	}
}

func (mw *JWTMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	mw.build(cfg)
	return mw.middleware()
}

// Shutdown no need to do anything
func (mw *JWTMiddleware) Shutdown(_ context.Context) error {
	return nil
}

func (mw *JWTMiddleware) middleware() gin.HandlerFunc {
	extractors, err := CreateExtractors(mw.Config.TokenLookup, mw.Config.AuthScheme)
	if err != nil {
		panic(err)
	}
	if len(mw.Config.TokenLookupFuncs) > 0 {
		extractors = append(mw.Config.TokenLookupFuncs, extractors...)
	}
	return func(c *gin.Context) {
		if mw.Config.Skipper(c) {
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
				token, err := mw.Config.ParseTokenFunc(c, authStr)
				if err != nil {
					lastTokenErr = err
					continue
				}
				if mw.TokenStore != nil {
					jti, ok := mw.Config.GetTokenIDFunc(token)
					if ok {
						if exists := mw.TokenStore.Has(c, jti); !exists {
							lastTokenErr = jwt.ErrTokenUnverifiable
							continue
						}
					} else {
						lastTokenErr = jwt.ErrTokenInvalidClaims
						continue
					}
				}
				// Store user information from token into context.
				if mw.Config.WithPrincipalContext != nil {
					err = mw.Config.WithPrincipalContext(c, token)
					if err != nil {
						lastTokenErr = err
						continue
					}
				}
				return
			}
		}
		err := lastTokenErr
		if err == nil {
			err = lastExtractorErr
		}
		if mw.Config.ErrorHandler != nil {
			err = mw.Config.ErrorHandler(c, err)
		}

		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err) //nolint:errcheck
		}
	}
}
