package authz

import (
	"fmt"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
)

// Options is the options for the authz middleware.
type Options struct {
	AppCode string `yaml:"appCode" json:"appCode"`
}

// Authorizer web api authorizer.
//
// Because of the dependency on JwtToken, the middleware configuration order must come after jwt.
type Authorizer struct{}

func New() *Authorizer {
	return &Authorizer{}
}

func Middleware() handler.Middleware {
	return New()
}

func (a *Authorizer) Name() string {
	return "authz"
}

func (a *Authorizer) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opt := Options{}
	if err := cfg.Unmarshal(&opt); err != nil {
		panic(err)
	}
	if security.DefaultAuthorizer == nil {
		panic("security.DefaultAuthorizer is nil")
	}
	return func(c *gin.Context) {
		gp, ok := security.GenericIdentityFromContext(c)
		if !ok {
			c.AbortWithError(http.StatusForbidden, fmt.Errorf("authorization failed: %s", "no identity found")) //nolint:errcheck
			return
		}
		res, err := security.DefaultAuthorizer.Conv(c, security.ArnRequestKindWeb, opt.AppCode, c.Request.Method, c.Request.URL.Path)
		if err != nil {
			c.AbortWithError(http.StatusForbidden, fmt.Errorf("authorization failed: %w", err)) //nolint:errcheck
			return
		}
		allowed, err := security.DefaultAuthorizer.Eval(c, gp, res)
		if err != nil {
			c.AbortWithError(http.StatusForbidden, fmt.Errorf("authorization failed: %w", err)) //nolint:errcheck
			return
		}
		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}
