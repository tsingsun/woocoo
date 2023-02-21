package authz

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/authz"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/security"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
)

// RestAuthorizer restful api authorizer.
//
// Because of the dependency on JwtToken, the middleware configuration order must come after jwt.
type RestAuthorizer struct {
	Authorization *authz.Authorization
}

func (a *RestAuthorizer) Name() string {
	return "rest-authz"
}

func (a *RestAuthorizer) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	if err := cfg.Unmarshal(a); err != nil {
		panic(err)
	}
	op := authz.SetDefaultAuthorization(cfg.Root())
	a.Authorization = op
	return func(c *gin.Context) {
		if !a.CheckPermission(c) {
			a.RequirePermission(c)
		}
	}
}

func (a *RestAuthorizer) Shutdown(ctx context.Context) error {
	return nil
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (a *RestAuthorizer) CheckPermission(c *gin.Context) (allowed bool) {
	var (
		err error
	)
	gp := security.GenericIdentityFromContext(c)
	pl, _ := c.Get(handler.AuthzContextKey)
	if pl != nil {
		for _, res := range pl.([]*security.PermissionItem) {
			allowed, err = a.Authorization.Enforcer.Enforce(gp.Name(), res.Action, res.Operator)
		}
	} else {
		allowed, err = a.Authorization.Enforcer.Enforce(gp.Name(), c.Request.URL.Path, c.Request.Method)
	}
	if err != nil {
		c.Error(err)
		return false
	}
	return allowed
}

// RequirePermission returns the 403 Forbidden to the client
func (a *RestAuthorizer) RequirePermission(c *gin.Context) {
	c.AbortWithStatus(http.StatusForbidden)
}
