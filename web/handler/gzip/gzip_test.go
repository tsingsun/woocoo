package gzip_test

import (
	nativeGzip "compress/gzip"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"testing"
)

const (
	testResponse        = "Gzip Test Response "
	testReverseResponse = "Gzip Test Reverse Response "
)

type rServer struct{}

func (s *rServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprint(rw, testReverseResponse)
}

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func newCloseNotifyingRecorder() *closeNotifyingRecorder {
	return &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func newServer(config map[string]any) *gin.Engine {
	// init reverse proxy server
	rServer := httptest.NewServer(new(rServer))
	target, _ := url.Parse(rServer.URL)
	rp := httputil.NewSingleHostReverseProxy(target)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	mid := gzip.NewGzip()
	router.Use(mid.ApplyFunc(conf.NewFromParse(conf.NewParserFromStringMap(config))))
	router.GET("/", func(c *gin.Context) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	router.Any("/reverse", func(c *gin.Context) {
		rp.ServeHTTP(c.Writer, c.Request)
	})
	return router
}

func TestConfig(t *testing.T) {
	assert.NotPanics(t, func() {
		newServer(map[string]any{
			"minSize": 1,
			"level":   -1,
		})
	})
	assert.Panics(t, func() {
		newServer(map[string]any{
			"minSize": 1,
			"level":   -5,
		})
	})
}

func TestGzip(t *testing.T) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Add("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	r := newServer(map[string]any{
		"minSize": 1,
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header().Get("Content-Encoding"), "gzip")
	assert.Equal(t, w.Header().Get("Vary"), "Accept-Encoding")
	assert.NotEqual(t, w.Header().Get("Content-Length"), "0")
	assert.NotEqual(t, w.Body.Len(), 19)
	assert.Equal(t, fmt.Sprint(w.Body.Len()), w.Header().Get("Content-Length"))

	wantBody, err := unzipBody(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, wantBody, testResponse)
}

func TestGzipPNG(t *testing.T) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/image.png", nil)
	req.Header.Add("Accept-Encoding", "gzip")

	router := gin.New()
	router.Use(gzip.NewGzip().ApplyFunc(conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
		"minSize": 1,
	}))))
	router.GET("/image.png", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "image/png")
		c.String(200, "this is a PNG!")
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header().Get("Content-Encoding"), "")
	assert.Equal(t, w.Header().Get("Vary"), "")
	assert.Equal(t, w.Body.String(), "this is a PNG!")
}

func TestExcludedExtensions(t *testing.T) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/index.html", nil)
	req.Header.Add("Accept-Encoding", "gzip")

	router := gin.New()
	mid := gzip.NewGzip()
	router.Use(mid.ApplyFunc(conf.NewFromParse(conf.NewParserFromStringMap(map[string]any{
		"minSize":            1,
		"excludedExtensions": []string{".html"},
	}))))
	router.GET("/index.html", func(c *gin.Context) {
		c.String(200, "this is a HTML!")
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "", w.Header().Get("Vary"))
	assert.Equal(t, "this is a HTML!", w.Body.String())
	assert.Equal(t, "", w.Header().Get("Content-Length"))
	assert.NoError(t, mid.Shutdown(context.Background()))
}

func unzipBody(r io.Reader) (string, error) {
	gr, err := nativeGzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gr.Close()
	body, _ := io.ReadAll(gr)
	return string(body), nil
}

func TestGzipWithReverseProxy(t *testing.T) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/reverse", nil)
	req.Header.Add("Accept-Encoding", "gzip")

	w := newCloseNotifyingRecorder()
	r := newServer(map[string]any{
		"minSize": 1,
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header().Get("Content-Encoding"), "gzip")
	assert.Equal(t, w.Header().Get("Vary"), "Accept-Encoding")
	assert.NotEqual(t, w.Header().Get("Content-Length"), "0")
	assert.NotEqual(t, w.Body.Len(), 19)
	assert.Equal(t, fmt.Sprint(w.Body.Len()), w.Header().Get("Content-Length"))

	wantBody, err := unzipBody(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, wantBody, testReverseResponse)
}

func TestGzipWithWeb(t *testing.T) {
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - gzip:
                minSize: 20
                excludedExtensions: [".html"]
                level: 1
`

	cfg := conf.NewFromBytes([]byte(cfgStr))
	srv := web.New(web.WithConfiguration(cfg.Sub("web")))
	srv.Router().GET("/", func(c *gin.Context) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.String(200, testResponse)
	})
	t.Run("lt minSize", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
		req.Header.Add("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		assert.Equal(t, w.Code, 200)
		assert.NotZero(t, w.Header().Get("Content-Length"))
		t.Log(w.Header().Get("Content-Length"))
		got, _ := io.ReadAll(w.Body)
		assert.Equal(t, testResponse, string(got))
	})
	t.Run("gt minSize", func(t *testing.T) {
		srv.Router().GET("/usezip", func(c *gin.Context) {
			gt := testResponse + "great than minSize"
			c.String(200, gt)
		})
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/usezip", nil)
		req.Header.Add("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		assert.Equal(t, w.Code, 200)
		assert.NotZero(t, w.Header().Get("Content-Length"))
		wantBody, err := unzipBody(w.Body)
		assert.NoError(t, err)
		assert.Contains(t, wantBody, "great than minSize")
	})
	t.Run("miss header no zip", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		assert.Equal(t, w.Code, 200)
		assert.Equal(t, w.Header().Get("Content-Encoding"), "")
		assert.Equal(t, w.Header().Get("Content-Length"), "19")
		assert.Equal(t, w.Body.String(), testResponse)
	})
	t.Run("json", func(t *testing.T) {
		srv.Router().GET("/json", func(c *gin.Context) {
			c.JSON(200, []gin.H{
				{"message": testResponse, "int": 1, "bool": true},
				{"message": testResponse, "int": 2, "bool": true},
				{"message": testResponse, "int": 3, "bool": true},
			},
			)
		})
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/json", nil)
		req.Header.Add("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		assert.Equal(t, w.Code, 200)
		assert.Equal(t, w.Header().Get("Content-Encoding"), "gzip")
		wantBody, err := unzipBody(w.Body)
		assert.NoError(t, err)
		assert.Contains(t, wantBody, testResponse)
	})
}

func BenchmarkGzip_S2k(b *testing.B) { benchmarkGzip(b, false, 2048, nativeGzip.DefaultCompression) }
func BenchmarkGzip_P2k(b *testing.B) { benchmarkGzip(b, true, 2048, nativeGzip.DefaultCompression) }
func BenchmarkGzip_S100k(b *testing.B) {
	benchmarkGzip(b, false, 102400, nativeGzip.DefaultCompression)
}
func BenchmarkGzip_P100k(b *testing.B) { benchmarkGzip(b, true, 102400, nativeGzip.DefaultCompression) }

func benchmarkGzip(b *testing.B, parallel bool, size, level int) {
	bin, err := os.ReadFile("../../../test/testdata/gzip/benchmark.json")
	if err != nil {
		b.Fatal(err)
	}
	var cfgStr = `
web:
  server:
  engine:
    routerGroups:
      - default:
          middlewares:
            - gzip:
                level: %d
`

	cfg := conf.NewFromBytes([]byte(fmt.Sprintf(cfgStr, level)))

	srv := web.New(web.WithConfiguration(cfg.Sub("web")))
	srv.Router().GET("/", func(c *gin.Context) {
		c.Header("Content-Length", strconv.Itoa(len(testResponse)))
		c.Writer.Write(bin[:size])
	})
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Add("Accept-Encoding", "gzip")
	b.ReportAllocs()
	b.SetBytes(int64(size))
	var run = func() {
		res := httptest.NewRecorder()
		srv.Router().ServeHTTP(res, req)
		if code := res.Code; code != 200 {
			b.Fatalf("Expected 200 but got %d", code)
		} else if blen := res.Body.Len(); blen < 500 {
			b.Fatalf("Expected complete response body, but got %d bytes", blen)
		}
	}
	if parallel {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				run()
			}
		})
	} else {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			run()
		}
	}
}
