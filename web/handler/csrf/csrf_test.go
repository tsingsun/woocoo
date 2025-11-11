package csrf

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
)

func TestApply(t *testing.T) {
	t.Run("empty cookie", func(t *testing.T) {
		mdl := CSRF()
		mdl.ApplyFunc(conf.NewFromStringMap(map[string]any{
			"authKey": "test",
		}))
		assert.Equal(t, "csrf", mdl.Name())
	})
	t.Run("err sameSite", func(t *testing.T) {
		mw := CSRF().(*Middleware)
		mw.ApplyFunc(conf.NewFromStringMap(map[string]any{
			"authKey": "test",
			"cookie": map[string]any{
				"name":     "test",
				"path":     "/",
				"domain":   "localhost",
				"maxAge":   3600,
				"secure":   true,
				"httpOnly": true,
				"sameSite": "2",
			},
		}))
		assert.True(t, mw.config.Cookie.Secure)
	})
	t.Run("err sameSite", func(t *testing.T) {
		assert.Panics(t, func() {
			CSRF().ApplyFunc(conf.NewFromStringMap(map[string]any{
				"authKey": "test",
				"cookie": map[string]any{
					"name":     "test",
					"path":     "/",
					"domain":   "localhost",
					"maxAge":   3600,
					"secure":   false,
					"httpOnly": true,
					"sameSite": "lax",
				},
			}))
		})
	})
}

func TestCSRF(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	w := httptest.NewRecorder()
	_, srv := gin.CreateTestContext(w)
	t.Run("empty", func(t *testing.T) {
		g := srv.Group("empty")
		g.Use(CSRF().ApplyFunc(conf.NewFromStringMap(map[string]any{
			"authKey": "test",
		})))
		g.GET("/", func(c *gin.Context) {
			c.String(200, "test")
		})
		w = httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/empty/", nil)
		srv.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), "test")
		assert.NotEmpty(t, w.Header().Get("Set-Cookie"))
	})
	t.Run("cookie", func(t *testing.T) {
		srv.Use(CSRF().ApplyFunc(conf.NewFromStringMap(map[string]any{
			"authKey":        "test",
			"trustedOrigins": []string{"localhost"},
			"cookie": map[string]any{
				"name":     "test",
				"path":     "/",
				"domain":   "localhost",
				"maxAge":   3600,
				"secure":   false,
				"httpOnly": true,
				"sameSite": "2",
			},
		})))
		srv.GET("/cookie/", func(c *gin.Context) {
			c.String(200, "test")
		})
		srv.POST("/cookie/", func(c *gin.Context) {
			c.String(200, "upload")
		})

		r := httptest.NewRequest("GET", "/cookie/", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), "test")
		token := w.Header().Get("X-CSRF-Token")
		assert.NotEmpty(t, token)

		gck := w.Header().Get("Set-Cookie")
		r = httptest.NewRequest("POST", "/cookie/", nil)
		r.Header.Set("Cookie", gck)
		r.Header.Set("X-CSRF-Token", token)
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		assert.Equal(t, 403, w.Code, "origin or referer check not pass")

		r = httptest.NewRequest("POST", "/cookie/", nil)
		r.Host = "localhost"
		r.Header.Set("Cookie", gck)
		r.Header.Set("Origin", "http://localhost/")
		r.Header.Set("X-CSRF-Token", token)
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)

		ps := strings.Split(gck, ";")
		ps[0] = "_gorilla_csrf=" + "wrong"
		wrongCookie := strings.Join(ps, ";")
		r = httptest.NewRequest("POST", "/cookie/", nil)
		r.Header.Set("Cookie", wrongCookie)
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		assert.Equal(t, 403, w.Code)
	})
	t.Run("skip", func(t *testing.T) {
		srv.Use(CSRF().ApplyFunc(conf.NewFromStringMap(map[string]any{
			"authKey": "test",
			"exclude": []string{"/skip"},
		})))
		srv.GET("/skip", func(c *gin.Context) {
			c.String(200, "skip")
		})
		w = httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/skip", nil)
		srv.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), "skip")
	})
}
