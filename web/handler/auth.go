package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/user"
	jwt "github.com/tsingsun/woocoo/third_party/appleboy/gin-jwt"
	"strings"
)

var (
	authConfigPath = strings.Join([]string{"web", "engine", "handleFuncs", "auth"}, conf.KeyDelimiter)
)

func DefaultAuthMiddleware(cfg *conf.Configuration) *jwt.GinJWTMiddleware {
	OrgIdHeader := "X-Org-Id"
	if k := cfg.String(strings.Join([]string{authConfigPath, "tenantHeader"}, conf.KeyDelimiter)); k != "" {
		OrgIdHeader = k
	}
	ac := &jwt.GinJWTMiddleware{
		Realm:       "woocoo",
		IdentityKey: "user",
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &user.User{
				"ID":    claims[jwt.IdentityKey].(string),
				"orgID": c.Request.Header.Get(OrgIdHeader),
			}
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"errors": []map[string]interface{}{
					{
						"code":    code,
						"message": message,
					},
				},
			})
		},
	}
	if ac.PrivKeyFile != "" {
		ac.PrivKeyFile = cfg.Abs(ac.PrivKeyFile)
	}
	if ac.PubKeyFile != "" {
		ac.PubKeyFile = cfg.Abs(ac.PubKeyFile)
	}
	if err := cfg.Parser().UnmarshalByJson(authConfigPath, ac); err != nil {
		panic(err)
	}
	ac.Key = []byte(cfg.String(strings.Join([]string{authConfigPath, "secret"}, conf.KeyDelimiter)))
	authMiddleware, err := jwt.New(ac)
	if err != nil {
		panic(err)
	}
	return authMiddleware
}

//AuthHandler jwt Token
//secret: map to Key
func AuthHandler(cnf *conf.Configuration) gin.HandlerFunc {
	return DefaultAuthMiddleware(cnf).MiddlewareFunc()
}
