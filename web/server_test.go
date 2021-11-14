package web_test

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/web"
	_ "github.com/tsingsun/woocoo/web/handler/gql"
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

func TestServer_Apply(t *testing.T) {
	srv := web.New()
	srv.Apply(cnf, "web")
}
