package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/user"
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf = testdata.Config
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
