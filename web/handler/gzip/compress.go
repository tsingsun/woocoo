package gzip

import (
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/gzhttp/writer"
)

// ResponseWriter is a wrapper for the http.ResponseWriter that compresses
type ResponseWriter struct {
	gin.ResponseWriter
	gzipWriter writer.GzipWriter
	minSize    int
	level      int
	useZip     bool
}

func (r *ResponseWriter) Close() (err error) {
	if r.useZip {
		err = r.gzipWriter.Close()
	}
	r.gzipWriter = nil
	return err
}

func (r *ResponseWriter) Write(data []byte) (int, error) {
	r.useZip = len(data) >= r.minSize
	if !r.useZip {
		return r.ResponseWriter.Write(data)
	}
	r.ResponseWriter.Header().Del("Content-Length")
	r.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	return r.gzipWriter.Write(data)
}

func (r *ResponseWriter) WriteHeader(code int) {
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseWriter) WriteString(s string) (int, error) {
	return r.Write([]byte(s))
}
