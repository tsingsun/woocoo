package authz

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/authz"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"net/http"
)

// Options is the options for the authz middleware.
type Options struct {
	AppCode       string
	Authorization *authz.Authorization
}

// Authorizer web api authorizer.
//
// Because of the dependency on JwtToken, the middleware configuration order must come after jwt.
type Authorizer struct {
}

func New() *Authorizer {
	return &Authorizer{}
}

func (a *Authorizer) Name() string {
	return "authz"
}

func (a *Authorizer) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opt := Options{}
	if err := cfg.Unmarshal(&opt); err != nil {
		panic(err)
	}
	opt.Authorization = authz.DefaultAuthorization
	if opt.Authorization == nil {
		var err error
		opt.Authorization, err = authz.NewAuthorization(cfg.Root())
		if err != nil {
			panic(fmt.Errorf("[web]authz: %w", err))
		}
	}
	return func(c *gin.Context) {
		gp := security.GenericIdentityFromContext(c)
		allowed, err := opt.Authorization.CheckPermission(c, gp, &security.PermissionItem{
			AppCode:  opt.AppCode,
			Action:   c.Request.URL.Path,
			Operator: c.Request.Method,
		})
		if err != nil {
			c.Error(fmt.Errorf("authorization failed: %w", err))
			return
		}
		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

func (a *Authorizer) Shutdown(ctx context.Context) error {
	return nil
}
