package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	t.Run("default", func(t *testing.T) {
		router := gin.New()
		cnf := conf.NewFromStringMap(map[string]any{})
		router.Use(CORS().ApplyFunc(cnf))
		router.GET("/", func(context *gin.Context) {
			return
		})
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Access-Control-Allow-Origin", "https://github.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("origin_not_allow", func(t *testing.T) {
		router := gin.New()
		cnf := conf.NewFromStringMap(map[string]any{
			"allowOrigins": []string{"https://woocoo.com"},
		})
		router.Use(CORS().ApplyFunc(cnf))
		router.GET("/", func(context *gin.Context) {
			return
		})
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Origin", "https://github.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
	t.Run("clear config headers", func(t *testing.T) {
		router := gin.New()
		cnf := conf.NewFromStringMap(map[string]any{
			"allowHeaders": "",
		})
		router.Use(CORS().ApplyFunc(cnf))
		router.GET("/", func(context *gin.Context) {
			return
		})
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Origin", "https://github.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
