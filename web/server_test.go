package web_test

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/test/testdata"
	"github.com/tsingsun/woocoo/web"
	"testing"
)

var cnf = testdata.Config

func init() {
	gin.SetMode(gin.ReleaseMode)
}
func TestServer_Apply(t *testing.T) {
	srv := web.New()
	srv.Apply(cnf, "web")
}
