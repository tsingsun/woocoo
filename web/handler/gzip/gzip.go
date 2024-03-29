package gzip

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/gzhttp"
	"github.com/klauspost/compress/gzhttp/writer"
	"github.com/klauspost/compress/gzhttp/writer/gzkp"
	"github.com/klauspost/compress/gzip"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	defaultExcludedExtensions = []string{".png", "gif", "jpg", "jpeg"}

	gwPool = sync.Pool{
		New: func() any {
			return &ResponseWriter{}
		},
	}
)

// Config is the configuration for the gzip middleware.
// Support min size and level setting
type Config struct {
	// MinSize is the minimum size to trigger compression. Default is 1024 bytes(1KB).
	MinSize int `json:"minSize" yaml:"minSize"`
	// gzip.DefaultCompression is used if Level is not set.
	// 0 : gzip.NoCompression; 1 - 9 : gzip.BestSpeed - gzip.BestCompression; -1 : gzip.DefaultCompression; -2 : gzip.HuffmanOnly
	Level int `json:"level" yaml:"level"`
	// ExcludedExtensions is a list of file extensions to skip compressing.
	ExcludedExtensions []string `json:"excludedExtensions" yaml:"excludedExtensions"`

	writerFactory         writer.GzipWriterFactory
	excludedExtensionsMap map[string]bool
}

func (c *Config) validate() error {
	min, max := c.writerFactory.Levels()
	if c.Level < min || c.Level > max {
		return fmt.Errorf("invalid compression level requested: %d, valid range %d -> %d", c.Level, min, max)
	}

	if c.MinSize < 0 {
		return fmt.Errorf("minimum size must be more than zero")
	}

	return nil
}

func convertToMap(slice []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range slice {
		m[s] = true
	}
	return m
}

// Middleware is a gzip handler
type Middleware struct {
	writerFactory writer.GzipWriterFactory
}

func Gzip() handler.Middleware {
	mw := NewGzip()
	return mw
}

// NewGzip returns a new gzip middleware.
func NewGzip() *Middleware {
	return &Middleware{
		writerFactory: writer.GzipWriterFactory{
			Levels: gzkp.Levels,
			New:    gzkp.NewWriter,
		},
	}
}

// Name returns the name of the middleware.
func (h *Middleware) Name() string {
	return "gzip"
}

func (h *Middleware) ApplyFunc(cfg *conf.Configuration) gin.HandlerFunc {
	opt := Config{
		Level:              gzip.DefaultCompression,
		MinSize:            gzhttp.DefaultMinSize,
		writerFactory:      h.writerFactory,
		ExcludedExtensions: defaultExcludedExtensions,
	}
	if err := cfg.Unmarshal(&opt); err != nil {
		panic(err)
	}
	if err := opt.validate(); err != nil {
		panic(err)
	}
	opt.excludedExtensionsMap = convertToMap(opt.ExcludedExtensions)
	return func(c *gin.Context) {
		if !h.shouldCompress(c.Request, opt) {
			return
		}
		gw := gwPool.Get().(*ResponseWriter)
		gw.ResponseWriter = c.Writer
		gw.gzipWriter = opt.writerFactory.New(c.Writer, opt.Level)
		gw.minSize = opt.MinSize
		c.Writer = gw
		c.Header("Vary", "Accept-Encoding")
		defer func() {
			gw.Close()
			if c.Writer.Size() > 0 { // 304 unmodified, size == -1
				c.Header("Content-Length", strconv.Itoa(c.Writer.Size()))
			}
			gwPool.Put(gw)
		}()
		c.Next()
	}
}

// Shutdown gzip noting to do here
func (h *Middleware) Shutdown(_ context.Context) error {
	return nil
}

// shouldCompress returns true if the given HTTP request indicates that it will
// accept a gzipped response.
func (h *Middleware) shouldCompress(req *http.Request, opt Config) bool {
	if req.Method == http.MethodHead || !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	if _, ok := opt.excludedExtensionsMap[filepath.Ext(req.URL.Path)]; ok {
		return false
	}
	return true
}
