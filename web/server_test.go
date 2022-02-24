package web_test

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/web"
	_ "github.com/tsingsun/woocoo/web/handler/gql"
	"net/http/httptest"
	"testing"
)

var cnf = testdata.Config

var logo = `
 ___      _______________________________ 
__ | /| / /  __ \  __ \  ___/  __ \  __ \
__ |/ |/ // /_/ / /_/ / /__ / /_/ / /_/ /
____/|__/ \____/\____/\___/ \____/\____/ 
`

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestNew(t *testing.T) {
	srv := web.New()
	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	srv.Router().Engine.GET("/user/:id", func(c *gin.Context) {
		c.String(200, "User")
	})
	srv.Router().Engine.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
}

func TestServer_Apply(t *testing.T) {
	srv := web.NewBuiltIn(web.Configuration(cnf))
	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	srv.Router().GET("/user/:id", func(c *gin.Context) {
		c.String(200, "User")
	})
	srv.Router().ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)

}
