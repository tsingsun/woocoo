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
		// LogoutHandler defines a function which is executed for user logout system.It clears something like cache.
		LogoutHandler func(*gin.Context)
		// ErrorHandler defines a function which is executed for an invalid token.
		// It may be used to define a custom JWT error and abort the request.like use:
		//  c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		ErrorHandler func(c *gin.Context, err error) error
		// TokenStoreKey is the name of the cache driver, default is "redis".
		// When this option is used, requirements: token cache KEY that uses the JWT ID.
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
	Config *JWTConfig
	// tokenStore is the cache for store token key.
	tokenStore cache.Cache
}

// NewJWT constructs a new JWT struct with supplied options.
func NewJWT(opts ...MiddlewareOption) *JWTMiddleware {
	mw := &JWTMiddleware{
		Config: &JWTConfig{
			JWTOptions: *auth.NewJWTOptions(),
		},
	}
	mipts := NewMiddlewareOption(opts...)
	if mipts.ConfigFunc != nil {
		mipts.ConfigFunc(mw.Config)
	}

	if mw.Config.WithPrincipalContext == nil {
		mw.Config.WithPrincipalContext = func(c *gin.Context, token *jwt.Token) error {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return auth.ErrJWTClaims
			}
			prpl := security.NewGenericPrincipalByClaims(claims)
			DerivativeContextWithValue(c, security.UserContextKey, prpl)
			return nil
		}
	}
	if mw.Config.LogoutHandler == nil {
		mw.Config.LogoutHandler = func(c *gin.Context) {
			gp, ok := security.GenericPrincipalFromContext(GetDerivativeContext(c))
			if !ok {
				return
			}
			cl := gp.Identity().Claims()
			jti, ok := mw.Config.GetTokenIDFunc(&jwt.Token{Claims: cl})
			if ok && mw.tokenStore != nil {
				if err := mw.tokenStore.Del(c, jti); err != nil {
					c.Error(err) // nolint: errcheck
				}
			}
		}
	}
	return mw
}

// JWT is the jwt middleware apply function. see MiddlewareNewFunc
func JWT() Middleware {
	return NewJWT()
}

func (mw *JWTMiddleware) Name() string {
	return JWTName
}

func (mw *JWTMiddleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	var err error
	if err = cfg.Unmarshal(mw.Config); err != nil {
		panic(err)
	}
	if err = mw.Config.Init(); err != nil {
		panic(err)
	}

	if mw.Config.TokenStoreKey != "" {
		if mw.tokenStore, err = cache.GetCache(mw.Config.TokenStoreKey); err != nil {
			panic(err)
		}
	}
	if mw.Config.Skipper == nil {
		mw.Config.Skipper = PathSkipper(mw.Config.Exclude)
	}
	return mw.middleware()
}

// HandleToken parse and check the token string.
// You can call it when do out of gin context
func (mw *JWTMiddleware) HandleToken(ctx context.Context, authStr string) (token *jwt.Token, err error) {
	token, err = mw.Config.ParseTokenFunc(ctx, authStr)
	if err != nil {
		return
	}
	if mw.tokenStore != nil {
		jti, ok := mw.Config.GetTokenIDFunc(token)
		if ok {
			if exists := mw.tokenStore.Has(ctx, jti); !exists {
				err = jwt.ErrTokenUnverifiable
				return
			}
		} else {
			err = jwt.ErrTokenInvalidClaims
			return
		}
	}
	return
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
		var (
			lastExtractorErr error
			lastTokenErr     error
		)
	outer:
		for _, extractor := range extractors {
			auths, err := extractor(c)
			if err != nil {
				lastExtractorErr = auth.ErrJWTMissing // backwards compatibility: all extraction errors are same (unlike KeyAuth)
				continue
			}
			for _, authStr := range auths {
				token, err := mw.HandleToken(c, authStr)
				if err != nil {
					lastTokenErr = err
					continue
				}
				// Store user information from token into context.
				if mw.Config.WithPrincipalContext != nil {
					if err := mw.Config.WithPrincipalContext(c, token); err != nil {
						lastTokenErr = err
						break outer
					}
				}
				return
			}
		}
		// must happen error
		if lastTokenErr == nil {
			lastTokenErr = lastExtractorErr
		}
		if mw.Config.ErrorHandler != nil {
			lastTokenErr = mw.Config.ErrorHandler(c, lastTokenErr)
		}

		if lastTokenErr != nil {
			c.AbortWithError(http.StatusUnauthorized, lastTokenErr) //nolint:errcheck
		}
	}
}
