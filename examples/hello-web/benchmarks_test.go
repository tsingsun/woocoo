package main_test

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
)

func BenchmarkLoggerMiddleware(B *testing.B) {
	var cfgstr = `
log:
  cores:
    - level: debug
      disableCaller: true
      disableStacktrace: true
      outputPaths: 
        - "logs/log-bench.log"
      errorOutputPaths:
        - stderr
  rotate:
    maxSize: 1
    maxage: 1
    maxbackups: 3
    localtime: true
    compress: false
web:
  server:
    addr: 0.0.0.0:33333
  engine:
    routerGroups:
      - default:
          middlewares:
            - accessLog:
            - recovery:
`
	_, currentFile, _, _ := runtime.Caller(0)
	basedir := filepath.Dir(currentFile)
	cfg := conf.NewFromBytes([]byte(cfgstr))
	cfg.SetBaseDir(basedir)
	log.NewBuiltIn()
	httpSvr := web.New(web.Configuration(cfg.Sub("web")))
	router := httpSvr.Router().Engine
	router.GET("/", func(c *gin.Context) {
		c.String(200, "hello world")
	})

	runRequest(B, router, "GET", "/")
	log.Global().Sync()
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

func runRequest(B *testing.B, router *gin.Engine, method, path string) {
	// create fake request
	//req, err := http.NewRequest(method, path, nil)
	//r := httptest.NewRequest(method, path, nil)
	//w := httptest.NewRecorder()
	r, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	w := newMockWriter()

	B.ReportAllocs()
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		router.ServeHTTP(w, r)
	}
}
