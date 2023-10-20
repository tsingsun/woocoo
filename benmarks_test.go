package woocoo

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"testing"
)

func BenchmarkGinDefault(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	runRequest(b, router, "GET", "/ping")
}

func BenchmarkWooCooWebDefault(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := web.New().Router().Engine
	router.Use(handler.NewAccessLog().ApplyFunc(conf.New()))
	router.Use(handler.Recovery().ApplyFunc(nil))
	router.GET("ping/", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	runRequest(b, router, "GET", "/ping")
}

func BenchmarkGinDefaultMockLogger(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithWriter(newMockWriter()))
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	runRequest(b, router, "GET", "/ping")
}

type mockWriter struct {
	headers http.Header
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		http.Header{},
	}
}

func (m *mockWriter) Header() (h http.Header) {
	return m.headers
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockWriter) WriteHeader(int) {}

func runRequest(b *testing.B, r *gin.Engine, method, path string) {
	// create fake request
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	w := newMockWriter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}
