package csrf

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
)

type Config struct {
	Skipper handler.Skipper
	// Exclude is a list of http paths to exclude from JWT auth
	//
	// path format must same as url.URL.Path started with "/" and ended with "/"
	Exclude []string `json:"exclude" yaml:"exclude"`
	// TokenLength is the length of the generated token.
	// Optional. Default value 32.
	TokenLength uint8 `yaml:"tokenLength"`

	AuthKey string `json:"authKey" yaml:"authKey"`
	// Default: X-CSRF-Token
	RequestHeader string        `json:"requestHeader" yaml:"requestHeader"`
	Cookie        *CookieConfig `json:"cookie" yaml:"cookie"`
}

type CookieConfig struct {
	Name     string        `json:"name" yaml:"name"`
	Path     string        `json:"path" yaml:"path"`
	Domain   string        `json:"domain" yaml:"domain"`
	MaxAge   int           `json:"maxAge" yaml:"maxAge"`
	Secure   bool          `json:"secure" yaml:"secure"`
	HttpOnly bool          `json:"httpOnly" yaml:"httpOnly"`
	SameSite http.SameSite `json:"sameSite" yaml:"sameSite"`
}

// Middleware implements a Cross-Site Request Forgery (CSRF) protection middleware.
type Middleware struct {
	config *Config
}

func NewMiddleware() *Middleware {
	mw := &Middleware{
		config: &Config{
			TokenLength:   32,
			RequestHeader: "X-CSRF-Token",
		},
	}
	return mw
}

func CSRF() handler.Middleware {
	return NewMiddleware()
}

func (mw *Middleware) Name() string {
	return "csrf"
}

// ServeHTTP empty implementation for csrf.Protect
func (mw *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Add(mw.config.RequestHeader, csrf.Token(r))
	}
}

func (mw *Middleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if err := cfg.Unmarshal(&mw.config); err != nil {
		panic(err)
	}
	if mw.config.Skipper == nil {
		mw.config.Skipper = handler.PathSkipper(mw.config.Exclude)
	}

	var opts []csrf.Option
	if mw.config.RequestHeader != "" {
		opts = append(opts, csrf.RequestHeader(mw.config.RequestHeader))
	}
	if mw.config.Cookie != nil {
		if mw.config.Cookie.Name != "" {
			opts = append(opts, csrf.CookieName(mw.config.Cookie.Name))
		}
		if mw.config.Cookie.Domain != "" {
			opts = append(opts, csrf.Domain(mw.config.Cookie.Domain))
		}
		if mw.config.Cookie.Path != "" {
			opts = append(opts, csrf.Path(mw.config.Cookie.Path))
		}
		if mw.config.Cookie.MaxAge != 0 {
			opts = append(opts, csrf.MaxAge(mw.config.Cookie.MaxAge))
		}
		if mw.config.Cookie.Secure {
			opts = append(opts, csrf.Secure(mw.config.Cookie.Secure))
		}
		if mw.config.Cookie.HttpOnly {
			opts = append(opts, csrf.HttpOnly(mw.config.Cookie.HttpOnly))
		}
		if mw.config.Cookie.SameSite != 0 {
			opts = append(opts, csrf.SameSite(csrf.SameSiteMode(mw.config.Cookie.SameSite)))
		}
	}
	opts = append(opts, csrf.ErrorHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := r.Context().Value(gin.ContextKey).(*gin.Context)
			c.AbortWithError(http.StatusForbidden, r.Context().Value("gorilla.csrf.Error").(error))
		})),
	)
	protect := csrf.Protect([]byte(mw.config.AuthKey), opts...)
	h := protect(mw)
	return func(c *gin.Context) {
		if mw.config.Skipper(c) {
			return
		}
		r := c.Request.WithContext(context.WithValue(c.Request.Context(), gin.ContextKey, c))
		h.ServeHTTP(c.Writer, r)
	}
}
