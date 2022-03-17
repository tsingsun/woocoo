package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/user"
	"testing"
)

func TestAuthHandler(t *testing.T) {
	IdentityHandler = func(c *gin.Context) interface{} {
		claims := ExtractClaims(c)
		return &user.User{
			"ID":    claims["sub"].(string),
			"orgID": c.Request.Header.Get("X-Org-Id"),
		}
	}
}
