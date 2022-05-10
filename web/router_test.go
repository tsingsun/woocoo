package web

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_AddRule(t *testing.T) {
	r := NewRouter(&ServerOptions{})
	r.Engine = gin.New()
	var newfunc gin.HandlerFunc = func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	}
	r.Engine.GET("/", newfunc)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.Engine.ServeHTTP(rec, req)
	assert.Equal(t, rec.Body.String(), "hello")
}
