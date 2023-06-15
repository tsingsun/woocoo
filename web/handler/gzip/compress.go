package gzip

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/gzhttp/writer"
)

// ResponseWriter is a wrapper for the http.ResponseWriter that compresses
type ResponseWriter struct {
	http.ResponseWriter
	gzipWriter writer.GzipWriter
	minSize    int
	level      int
}

func (r *ResponseWriter) Close() error {
	err := r.gzipWriter.Close()
	r.gzipWriter = nil
	r.ResponseWriter = nil
	return err
}

func (r *ResponseWriter) Write(data []byte) (int, error) {
	r.Header().Del("Content-Length")
	if len(data) < r.minSize {
		return r.ResponseWriter.Write(data)
	}
	r.Header().Set("Content-Encoding", "gzip")
	return r.gzipWriter.Write(data)
}

// GinResponseWriter is a wrapper response write using GZIP for Gin
type GinResponseWriter struct {
	gin.ResponseWriter
	gzipWriter *ResponseWriter
}

// WriteString writes the string into the response body.
func (g *GinResponseWriter) WriteString(s string) (int, error) {
	return g.gzipWriter.Write([]byte(s))
}

func (g *GinResponseWriter) Write(data []byte) (int, error) {
	return g.gzipWriter.Write(data)
}

func (g *GinResponseWriter) Flush() {
	g.Header().Del("Content-Length")
	g.gzipWriter.gzipWriter.Flush()
}

func (g *GinResponseWriter) WriteHeader(code int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}
